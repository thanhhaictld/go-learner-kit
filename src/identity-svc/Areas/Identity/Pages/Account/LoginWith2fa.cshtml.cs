using System.ComponentModel.DataAnnotations;
using System.Security.Claims;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class LoginWith2faModel(UserManager<ApplicationUser> users, SignInManager<ApplicationUser> signInManager) : PageModel
{
    [BindProperty] public InputModel Input { get; set; } = new();
    public sealed class InputModel { [Required, StringLength(7, MinimumLength = 6)] public string Code { get; set; } = ""; public bool RememberMachine { get; set; } }

    public async Task<IActionResult> OnGetAsync() => await GetPendingUserAsync() is null ? RedirectToPage("Login") : Page();

    public async Task<IActionResult> OnPostAsync()
    {
        if (!ModelState.IsValid) return Page();
        var pending = await PendingSignIn.GetAsync(HttpContext);
        var user = pending is null ? null : await users.FindByIdAsync(pending.UserId);
        if (user is null || pending?.OrganizationId is null) return RedirectToPage("Login");
        var code = Input.Code.Replace(" ", "", StringComparison.Ordinal).Replace("-", "", StringComparison.Ordinal);
        if (!await users.VerifyTwoFactorTokenAsync(user, users.Options.Tokens.AuthenticatorTokenProvider, code))
        {
            ModelState.AddModelError(string.Empty, "Invalid authenticator code.");
            return Page();
        }
        if (Input.RememberMachine) await signInManager.RememberTwoFactorClientAsync(user);
        await PendingSignIn.ClearAsync(HttpContext);
        await signInManager.SignInWithClaimsAsync(user, pending.RememberMe, [new Claim("org_id", pending.OrganizationId.Value.ToString())]);
        return LocalRedirect(pending.ReturnUrl);
    }

    private async Task<ApplicationUser?> GetPendingUserAsync()
    {
        var pending = await PendingSignIn.GetAsync(HttpContext);
        return pending is null ? null : await users.FindByIdAsync(pending.UserId);
    }
}
