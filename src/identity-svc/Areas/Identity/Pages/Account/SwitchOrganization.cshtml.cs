using System.Security.Claims;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class SwitchOrganizationModel(UserManager<ApplicationUser> users, SignInManager<ApplicationUser> signInManager, OrganizationService organizations) : PageModel
{
    [BindProperty] public Guid OrganizationId { get; set; }
    [BindProperty(SupportsGet = true)] public string? ReturnUrl { get; set; }
    public IReadOnlyList<Organization> Organizations { get; private set; } = [];
    public string CurrentUserName { get; private set; } = "";
    public string CurrentUserEmail { get; private set; } = "";

    public async Task<IActionResult> OnGetAsync(CancellationToken cancellationToken)
    {
        var user = await users.GetUserAsync(User);
        if (user is null) return RedirectToPage("Login", new { returnUrl = ReturnUrl });
        Organizations = await organizations.GetForUserAsync(user.Id, cancellationToken);
        CurrentUserName = user.UserName ?? user.Email ?? "User";
        CurrentUserEmail = user.Email ?? "";
        return Page();
    }

    public async Task<IActionResult> OnPostAsync(CancellationToken cancellationToken)
    {
        var user = await users.GetUserAsync(User);
        if (user is null) return RedirectToPage("Login", new { returnUrl = ReturnUrl });
        var available = await organizations.GetForUserAsync(user.Id, cancellationToken);
        if (available.All(x => x.Id != OrganizationId))
        {
            ModelState.AddModelError(string.Empty, "That organization is not available.");
            Organizations = available;
            CurrentUserName = user.UserName ?? user.Email ?? "User";
            CurrentUserEmail = user.Email ?? "";
            return Page();
        }
        await signInManager.SignInWithClaimsAsync(user, true, [new Claim("org_id", OrganizationId.ToString())]);
        return LocalRedirect(ReturnUrl ?? Url.Content("~/"));
    }
}
