using Identity.Svc.Data;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using OpenIddict.Abstractions;
using OpenIddict.Server.AspNetCore;

namespace Identity.Svc.Controllers;

public sealed class UserinfoController(UserManager<ApplicationUser> users) : Controller
{
    [Authorize(AuthenticationSchemes = OpenIddictServerAspNetCoreDefaults.AuthenticationScheme)]
    [HttpGet("~/connect/userinfo"), HttpPost("~/connect/userinfo")]
    public async Task<IActionResult> Userinfo()
    {
        var subject = User.GetClaim(OpenIddictConstants.Claims.Subject);
        if (string.IsNullOrEmpty(subject)) return Challenge(OpenIddictServerAspNetCoreDefaults.AuthenticationScheme);
        var user = await users.FindByIdAsync(subject);
        if (user is null) return Challenge(OpenIddictServerAspNetCoreDefaults.AuthenticationScheme);
        return Ok(new Dictionary<string, object?>
        {
            [OpenIddictConstants.Claims.Subject] = user.Id,
            [OpenIddictConstants.Claims.Email] = user.Email,
            [OpenIddictConstants.Claims.Name] = user.UserName,
            ["org_id"] = User.GetClaim("org_id")
        });
    }
}
