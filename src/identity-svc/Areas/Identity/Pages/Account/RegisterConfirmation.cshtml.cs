using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class RegisterConfirmationModel : PageModel
{
    [BindProperty(SupportsGet = true)] public string? Email { get; set; }
}
