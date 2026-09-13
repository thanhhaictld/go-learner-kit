using System.Security.Claims;
using Microsoft.AspNetCore.Authentication;
using Microsoft.AspNetCore.Authentication.Cookies;
using Microsoft.AspNetCore.Authentication.OpenIdConnect;
using Microsoft.IdentityModel.Tokens;
using Yarp.ReverseProxy.Transforms;
using Yarp.ReverseProxy.Transforms.Builder;

var builder = WebApplication.CreateBuilder(args);
var configuration = builder.Configuration;

builder.Services.AddAuthentication(options =>
    {
        options.DefaultScheme = CookieAuthenticationDefaults.AuthenticationScheme;
        options.DefaultChallengeScheme = OpenIdConnectDefaults.AuthenticationScheme;
    })
    .AddCookie(options =>
    {
        options.Cookie.Name = "saas-admin-portal";
        options.Cookie.HttpOnly = true;
        options.Cookie.SameSite = SameSiteMode.Lax;
        options.Cookie.SecurePolicy = CookieSecurePolicy.SameAsRequest;
    })
    .AddOpenIdConnect(options =>
    {
        options.Authority = configuration["Authentication:Authority"];
        options.ClientId = configuration["Authentication:ClientId"];
        options.ResponseType = "code";
        options.UsePkce = true;
        options.SaveTokens = false;
        options.RequireHttpsMetadata = configuration.GetValue<bool>("Authentication:RequireHttpsMetadata");
        options.MapInboundClaims = false;
        options.GetClaimsFromUserInfoEndpoint = true;
        options.TokenValidationParameters = new TokenValidationParameters
        {
            NameClaimType = "name",
            RoleClaimType = "role"
        };
        options.Scope.Clear();
        options.Scope.Add("openid");
        options.Scope.Add("profile");
        options.Scope.Add("email");
        options.Scope.Add("offline_access");
        var backchannelAuthority = configuration["Authentication:BackchannelAuthority"];
        if (Uri.TryCreate(options.Authority, UriKind.Absolute, out var publicAuthority) &&
            Uri.TryCreate(backchannelAuthority, UriKind.Absolute, out var internalAuthority))
            options.Backchannel = new HttpClient(new AuthorityRewriteHandler(publicAuthority, internalAuthority));
    });

builder.Services.AddAuthorizationBuilder()
    .AddPolicy("portal-user", policy => policy
        .RequireAuthenticatedUser()
        .RequireAssertion(context =>
            Guid.TryParse(context.User.FindFirstValue("sub"), out _) &&
            Guid.TryParse(context.User.FindFirstValue("org_id"), out _)));
builder.Services.AddReverseProxy()
    .LoadFromConfig(configuration.GetSection("ReverseProxy"))
    .AddTransforms<TrustedIdentityTransformProvider>();

var app = builder.Build();
app.UseExceptionHandler("/bff/error");
app.UseStaticFiles();
app.UseAuthentication();
app.UseAuthorization();

app.MapGet("/bff/login", (string? returnUrl) =>
{
    var redirectUri = IsLocalUrl(returnUrl) ? returnUrl! : "/";
    return Results.Challenge(new AuthenticationProperties { RedirectUri = redirectUri }, [OpenIdConnectDefaults.AuthenticationScheme]);
});

app.MapPost("/bff/logout", () =>
    Results.SignOut(new AuthenticationProperties { RedirectUri = "/" }, [CookieAuthenticationDefaults.AuthenticationScheme, OpenIdConnectDefaults.AuthenticationScheme]));

app.MapGet("/bff/user", (ClaimsPrincipal user) =>
{
    var subject = user.FindFirstValue("sub");
    var organizationId = user.FindFirstValue("org_id");
    if (!Guid.TryParse(subject, out _) || !Guid.TryParse(organizationId, out _)) return Results.Forbid();
    return Results.Ok(new
    {
        subject,
        name = user.FindFirstValue("name"),
        email = user.FindFirstValue("email"),
        organizationId
    });
}).RequireAuthorization("portal-user");

app.MapGet("/bff/ready", () => Results.Ok(new { status = "ok" }));
app.MapGet("/bff/error", () => Results.Problem("The portal is temporarily unavailable."));
app.MapReverseProxy();
app.MapFallbackToFile("index.html");
app.Run();

static bool IsLocalUrl(string? value) => !string.IsNullOrWhiteSpace(value) && value.StartsWith('/') && !value.StartsWith("//");

public sealed class TrustedIdentityTransformProvider : ITransformProvider
{
    public void ValidateRoute(TransformRouteValidationContext context) { }

    public void ValidateCluster(TransformClusterValidationContext context) { }

    public void Apply(TransformBuilderContext context)
    {
        if (!string.Equals(context.Route.RouteId, "users", StringComparison.Ordinal)) return;

        context.AddRequestTransform(transformContext =>
        {
            var subject = transformContext.HttpContext.User.FindFirstValue("sub");
            var organizationId = transformContext.HttpContext.User.FindFirstValue("org_id");
            if (!Guid.TryParse(subject, out _) || !Guid.TryParse(organizationId, out _))
            {
                transformContext.HttpContext.Response.StatusCode = StatusCodes.Status403Forbidden;
                return ValueTask.CompletedTask;
            }

            // Never permit browser-provided identity headers to reach trusted services.
            transformContext.ProxyRequest.Headers.Remove("X-User-ID");
            transformContext.ProxyRequest.Headers.Remove("X-Organization-ID");
            transformContext.ProxyRequest.Headers.TryAddWithoutValidation("X-User-ID", subject);
            transformContext.ProxyRequest.Headers.TryAddWithoutValidation("X-Organization-ID", organizationId);
            return ValueTask.CompletedTask;
        });
    }
}

public sealed class AuthorityRewriteHandler(Uri publicAuthority, Uri internalAuthority) : HttpClientHandler
{
    protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
    {
        if (request.RequestUri is { } uri &&
            string.Equals(uri.Host, publicAuthority.Host, StringComparison.OrdinalIgnoreCase) &&
            uri.Port == publicAuthority.Port)
            request.RequestUri = new Uri(internalAuthority, uri.PathAndQuery);
        return base.SendAsync(request, cancellationToken);
    }
}
