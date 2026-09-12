using System.ComponentModel.DataAnnotations;
using System.Security.Claims;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class LoginWithRecoveryCodeModel(UserManager<ApplicationUser> users, SignInManager<ApplicationUser> signInManager) : PageModel
{
    [BindProperty] public InputModel Input { get; set; } = new();
    public sealed class InputModel { [Required] public string RecoveryCode { get; set; } = ""; }
    public async Task<IActionResult> OnPostAsync()
    {
        if (!ModelState.IsValid) return Page();
        var pending = await PendingSignIn.GetAsync(HttpContext);
        var user = pending is null ? null : await users.FindByIdAsync(pending.UserId);
        if (user is null || pending?.OrganizationId is null) return RedirectToPage("Login");
        var result = await users.RedeemTwoFactorRecoveryCodeAsync(user, Input.RecoveryCode.Replace(" ", "", StringComparison.Ordinal));
        if (!result.Succeeded) { ModelState.AddModelError(string.Empty, "Invalid recovery code."); return Page(); }
        await PendingSignIn.ClearAsync(HttpContext);
        await signInManager.SignInWithClaimsAsync(user, pending.RememberMe, [new Claim("org_id", pending.OrganizationId.Value.ToString())]);
        return LocalRedirect(pending.ReturnUrl);
    }
}
