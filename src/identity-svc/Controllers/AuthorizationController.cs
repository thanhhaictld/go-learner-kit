using System.Security.Claims;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore;
using Microsoft.AspNetCore.Authentication;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using OpenIddict.Abstractions;
using OpenIddict.Server.AspNetCore;

namespace Identity.Svc.Controllers;

public sealed class AuthorizationController(UserManager<ApplicationUser> users, OrganizationService organizations) : Controller
{
    [Authorize]
    [HttpGet("~/connect/authorize"), HttpPost("~/connect/authorize")]
    public async Task<IActionResult> Authorize(CancellationToken cancellationToken)
    {
        var request = HttpContext.GetOpenIddictServerRequest() ?? throw new InvalidOperationException("The OpenID Connect request cannot be retrieved.");
        var user = await users.GetUserAsync(User) ?? throw new InvalidOperationException("The signed-in user no longer exists.");
        var memberships = await organizations.GetForUserAsync(user.Id, cancellationToken);
        if (memberships.Count == 0) return Forbid(OpenIddictServerAspNetCoreDefaults.AuthenticationScheme);

        var organizationId = User.FindFirstValue("org_id");
        if (memberships.Count > 1 || !Guid.TryParse(organizationId, out var activeOrganization) || memberships.All(x => x.Id != activeOrganization))
        {
            var returnUrl = Request.PathBase + Request.Path + Request.QueryString;
            return Redirect($"/Identity/Account/SwitchOrganization?returnUrl={Uri.EscapeDataString(returnUrl)}");
        }

        var identity = new ClaimsIdentity(OpenIddictServerAspNetCoreDefaults.AuthenticationScheme,
            OpenIddictConstants.Claims.Name, OpenIddictConstants.Claims.Role);
        identity.SetClaim(OpenIddictConstants.Claims.Subject, user.Id)
            .SetClaim(OpenIddictConstants.Claims.Email, user.Email)
            .SetClaim(OpenIddictConstants.Claims.Name, user.DisplayName)
            .SetClaim("org_id", organizationId)
            .SetScopes(request.GetScopes())
            .SetDestinations(GetDestinations);
        return SignIn(new ClaimsPrincipal(identity), OpenIddictServerAspNetCoreDefaults.AuthenticationScheme);
    }

    [HttpGet("~/connect/logout"), HttpPost("~/connect/logout")]
    public async Task<IActionResult> Logout()
    {
        await HttpContext.SignOutAsync(IdentityConstants.ApplicationScheme);
        return SignOut(OpenIddictServerAspNetCoreDefaults.AuthenticationScheme);
    }

    private static IEnumerable<string> GetDestinations(Claim claim)
    {
        yield return OpenIddictConstants.Destinations.AccessToken;
        if (claim.Type is OpenIddictConstants.Claims.Email or OpenIddictConstants.Claims.Name or "org_id")
            yield return OpenIddictConstants.Destinations.IdentityToken;
    }
}
