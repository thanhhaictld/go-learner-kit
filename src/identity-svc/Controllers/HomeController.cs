using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace Identity.Svc.Controllers;

[Authorize]
public sealed class HomeController : Controller
{
    [HttpGet("/")]
    public IActionResult Index() => Redirect("/Identity/Account/Manage");
}
