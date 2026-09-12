using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class OrganizationSettingsModel(UserManager<ApplicationUser> users, OrganizationService organizations, OpenFgaProvisioner provisioner) : PageModel
{
    [BindProperty(SupportsGet = true)] public Guid OrganizationId { get; set; }
    [BindProperty] public bool MfaRequired { get; set; }
    public Organization? Organization { get; private set; }
    public bool CanManage { get; private set; }

    public async Task<IActionResult> OnGetAsync(CancellationToken cancellationToken)
    {
        var user = await users.GetUserAsync(User);
        if (user is null) return Challenge();
        Organization = await organizations.GetForUserAsync(OrganizationId, user.Id, cancellationToken);
        if (Organization is null) return NotFound();
        CanManage = await CanManageAsync(user, cancellationToken);
        MfaRequired = Organization.MfaRequired;
        return Page();
    }

    public async Task<IActionResult> OnPostAsync(CancellationToken cancellationToken)
    {
        var user = await users.GetUserAsync(User);
        if (user is null) return Challenge();
        Organization = await organizations.GetForUserAsync(OrganizationId, user.Id, cancellationToken);
        if (Organization is null) return NotFound();
        CanManage = await CanManageAsync(user, cancellationToken);
        if (!CanManage) return Forbid();
        if (!await organizations.UpdateMfaPolicyAsync(OrganizationId, user.Id, MfaRequired, cancellationToken)) return Forbid();
        return RedirectToPage(new { OrganizationId });
    }

    private async Task<bool> CanManageAsync(ApplicationUser user, CancellationToken cancellationToken)
    {
        return await provisioner.CanManageOrganizationAsync(OrganizationId, user.Id, cancellationToken);
    }
}
