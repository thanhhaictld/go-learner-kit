using System.ComponentModel.DataAnnotations;
using System.Security.Claims;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class LoginModel(
    ILogger<LoginModel> logger,
    SignInManager<ApplicationUser> signInManager, UserManager<ApplicationUser> users, OrganizationService organizations) : PageModel
{
    private readonly ILogger<LoginModel> logger = logger;


    [BindProperty] public InputModel Input { get; set; } = new();
    [BindProperty(SupportsGet = true)] public string? ReturnUrl { get; set; }
    public sealed class InputModel { [Required, EmailAddress] public string Email { get; set; } = ""; [Required, DataType(DataType.Password)] public string Password { get; set; } = ""; public bool RememberMe { get; set; } }

    public void OnGet() => ReturnUrl ??= Url.Content("~/");

    public async Task<IActionResult> OnPostAsync(CancellationToken cancellationToken)
    {
        ReturnUrl ??= Url.Content("~/");
        if (!ModelState.IsValid) { logger.LogWarning("Invalid sign-in attempt for email: {Email}", Input.Email); return Page(); }
        ;
        var user = await users.FindByEmailAsync(Input.Email);
        if (user is null) { ModelState.AddModelError(string.Empty, "Invalid sign-in attempt."); logger.LogWarning("Invalid sign-in attempt for email: {Email}", Input.Email); return Page(); }
        var result = await signInManager.CheckPasswordSignInAsync(user, Input.Password, lockoutOnFailure: true);
        if (!result.Succeeded) { ModelState.AddModelError(string.Empty, "Invalid sign-in attempt."); logger.LogWarning("Invalid sign-in attempt for email: {Email}", Input.Email); return Page(); }
        var memberships = await organizations.GetForUserAsync(user.Id, cancellationToken);
        if (memberships.Count == 0) { ModelState.AddModelError(string.Empty, "This account is not a member of an organization."); logger.LogWarning("User {UserId} is not a member of any organization.", user.Id); return Page(); }
        if (memberships.Count > 1) return RedirectToPage("SwitchOrganization", new { returnUrl = ReturnUrl });
        await signInManager.SignInWithClaimsAsync(user, Input.RememberMe, [new Claim("org_id", memberships[0].Id.ToString())]);
        return LocalRedirect(ReturnUrl);
    }
}
