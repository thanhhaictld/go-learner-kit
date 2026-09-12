using System.ComponentModel.DataAnnotations;
using System.Security.Claims;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;

namespace Identity.Svc.Areas.Identity.Pages.Account.Manage;

public sealed class IndexModel(UserManager<ApplicationUser> users, SignInManager<ApplicationUser> signInManager, OrganizationService organizations) : PageModel
{
    [BindProperty] public InputModel Input { get; set; } = new();
    public IReadOnlyList<OrganizationMembership> Memberships { get; private set; } = [];
    public sealed class InputModel { [Required, StringLength(255)] public string DisplayName { get; set; } = ""; }

    public async Task<IActionResult> OnGetAsync(CancellationToken cancellationToken)
    {
        var user = await users.GetUserAsync(User);
        if (user is null) return Challenge();
        Input.DisplayName = user.DisplayName;
        Memberships = await organizations.GetMembershipsForUserAsync(user.Id, cancellationToken);
        return Page();
    }

    public async Task<IActionResult> OnPostAsync(CancellationToken cancellationToken)
    {
        var user = await users.GetUserAsync(User);
        if (user is null) return Challenge();
        if (!ModelState.IsValid) { Memberships = await organizations.GetMembershipsForUserAsync(user.Id, cancellationToken); return Page(); }
        user.DisplayName = Input.DisplayName.Trim();
        var result = await users.UpdateAsync(user);
        if (!result.Succeeded) { foreach (var error in result.Errors) ModelState.AddModelError(string.Empty, error.Description); Memberships = await organizations.GetMembershipsForUserAsync(user.Id, cancellationToken); return Page(); }
        var organizationId = User.FindFirstValue("org_id");
        if (Guid.TryParse(organizationId, out var organization))
            await signInManager.SignInWithClaimsAsync(user, false, [new Claim("org_id", organization.ToString())]);
        else
            await signInManager.RefreshSignInAsync(user);
        return RedirectToPage();
    }
}
