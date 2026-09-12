using System.Security.Claims;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace Identity.Svc.Controllers;

[Authorize]
public sealed class HomeController : Controller
{
    [HttpGet("/")]
    public IActionResult Index()
    {
        var claims = new Dictionary<string, string?>
        {
            ["sub"] = User.FindFirst(ClaimTypes.NameIdentifier)?.Value,
            ["email"] = User.FindFirst(ClaimTypes.Email)?.Value,
            ["name"] = User.Identity?.Name,
            ["org_id"] = User.FindFirst("org_id")?.Value
        };

        return View(claims.Where(claim => claim.Value is not null));
    }
}
