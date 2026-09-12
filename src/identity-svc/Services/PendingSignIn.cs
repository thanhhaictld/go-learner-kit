using System.Security.Claims;
using Microsoft.AspNetCore.Authentication;

namespace Identity.Svc.Services;

public sealed record PendingSignInInfo(string UserId, Guid? OrganizationId, bool RememberMe, string ReturnUrl);

public static class PendingSignIn
{
    public const string Scheme = "PendingSignIn";

    public static Task SetAsync(HttpContext context, string userId, Guid? organizationId, bool rememberMe, string returnUrl) =>
        context.SignInAsync(Scheme, new ClaimsPrincipal(new ClaimsIdentity(
        [new Claim(ClaimTypes.NameIdentifier, userId), new Claim("remember_me", rememberMe.ToString()), new Claim("return_url", returnUrl),
            .. (organizationId is null ? [] : new[] { new Claim("org_id", organizationId.Value.ToString()) })], Scheme)));

    public static async Task<PendingSignInInfo?> GetAsync(HttpContext context)
    {
        var result = await context.AuthenticateAsync(Scheme);
        if (!result.Succeeded || result.Principal is null) return null;
        var userId = result.Principal.FindFirstValue(ClaimTypes.NameIdentifier);
        var returnUrl = result.Principal.FindFirstValue("return_url");
        if (string.IsNullOrWhiteSpace(userId) || string.IsNullOrWhiteSpace(returnUrl)) return null;
        var orgText = result.Principal.FindFirstValue("org_id");
        return new PendingSignInInfo(userId, Guid.TryParse(orgText, out var organizationId) ? organizationId : null,
            bool.TryParse(result.Principal.FindFirstValue("remember_me"), out var rememberMe) && rememberMe, returnUrl);
    }

    public static Task ClearAsync(HttpContext context) => context.SignOutAsync(Scheme);
}
