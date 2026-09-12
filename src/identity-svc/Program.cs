using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.EntityFrameworkCore;
using OpenIddict.Abstractions;
using Prometheus;

var builder = WebApplication.CreateBuilder(args);
builder.Configuration.AddEnvironmentVariables();
var configuration = builder.Configuration;

builder.Services.AddDbContext<ApplicationDbContext>(options =>
    options.UseNpgsql(configuration.GetConnectionString("Default"), npgsql =>
        npgsql.MigrationsAssembly(typeof(Program).Assembly.FullName)));

builder.Services.AddIdentity<ApplicationUser, IdentityRole>(options =>
    {
        options.SignIn.RequireConfirmedAccount = true;
        options.Password.RequiredLength = 12;
        options.Password.RequireNonAlphanumeric = true;
        options.Lockout.MaxFailedAccessAttempts = 5;
    })
    .AddEntityFrameworkStores<ApplicationDbContext>()
    .AddDefaultTokenProviders();
builder.Services.ConfigureApplicationCookie(options =>
{
    options.LoginPath = "/Identity/Account/Login";
    options.Cookie.Name = "identity-svc";
});
builder.Services.AddAuthentication()
    .AddCookie("PendingSignIn", options =>
    {
        options.Cookie.Name = "identity-svc-pending";
        options.ExpireTimeSpan = TimeSpan.FromMinutes(10);
        options.SlidingExpiration = false;
    });
builder.Services.Configure<EmailOptions>(configuration.GetSection("Email"));
builder.Services.AddSingleton<IAccountEmailSender, SmtpAccountEmailSender>();

builder.Services.AddControllersWithViews();
builder.Services.AddRazorPages();
builder.Services.AddHttpClient<OpenFgaProvisioner>(client => client.Timeout = TimeSpan.FromSeconds(5));
builder.Services.AddScoped<OrganizationService>();
builder.Services.AddScoped<BootstrapService>();

builder.Services.AddOpenIddict()
    .AddCore(options => options.UseEntityFrameworkCore().UseDbContext<ApplicationDbContext>())
    .AddServer(options =>
    {
        options.SetIssuer(new Uri(configuration["OpenIddict:Issuer"] ?? "http://localhost:8081"));
        options.SetAuthorizationEndpointUris("connect/authorize")
            .SetTokenEndpointUris("connect/token")
            .SetUserinfoEndpointUris("connect/userinfo")
            .SetLogoutEndpointUris("connect/logout")
            .SetConfigurationEndpointUris(".well-known/openid-configuration")
            .SetCryptographyEndpointUris(".well-known/jwks");
        options.RegisterScopes(OpenIddictConstants.Scopes.Email, OpenIddictConstants.Scopes.Profile,
            OpenIddictConstants.Scopes.Roles, OpenIddictConstants.Scopes.OfflineAccess);
        options.AllowAuthorizationCodeFlow().AllowRefreshTokenFlow().AllowClientCredentialsFlow();
        options.RequireProofKeyForCodeExchange();
        options.AddDevelopmentEncryptionCertificate().AddDevelopmentSigningCertificate();
        options.DisableAccessTokenEncryption();
        var aspNetCore = options.UseAspNetCore();
        if (configuration.GetValue<bool>("OpenIddict:AllowInsecureHttp"))
            aspNetCore.DisableTransportSecurityRequirement();
        aspNetCore.EnableAuthorizationEndpointPassthrough()
            .EnableUserinfoEndpointPassthrough()
            .EnableLogoutEndpointPassthrough()
            .EnableStatusCodePagesIntegration();
    });

var app = builder.Build();

if (args.Contains("--migrate", StringComparer.Ordinal))
{
    await using var scope = app.Services.CreateAsyncScope();
    await scope.ServiceProvider.GetRequiredService<BootstrapService>().MigrateAndSeedAsync(CancellationToken.None);
    return;
}

app.UseExceptionHandler("/error");
app.UseStaticFiles();
app.UseRouting();
app.UseHttpMetrics();
app.UseAuthentication();
app.UseAuthorization();

app.MapControllers();
app.MapRazorPages();
app.MapGet("/health/live", () => Results.Ok(new { status = "ok" }));
app.MapGet("/health/ready", async (ApplicationDbContext db, CancellationToken cancellationToken) =>
    await db.Database.CanConnectAsync(cancellationToken) ? Results.Ok(new { status = "ok" }) : Results.StatusCode(StatusCodes.Status503ServiceUnavailable));
app.MapMetrics();

app.Run();

public partial class Program;
