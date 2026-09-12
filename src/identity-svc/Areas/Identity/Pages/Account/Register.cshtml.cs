using System.ComponentModel.DataAnnotations;
using System.Security.Claims;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class RegisterModel(UserManager<ApplicationUser> users, SignInManager<ApplicationUser> signInManager, OrganizationService organizations) : PageModel
{
    [BindProperty] public InputModel Input { get; set; } = new();
    [BindProperty(SupportsGet = true)] public string? ReturnUrl { get; set; }
    public sealed class InputModel
    {
        [Required, EmailAddress] public string Email { get; set; } = "";
        [Required, DataType(DataType.Password), StringLength(100, MinimumLength = 12)] public string Password { get; set; } = "";
        [Required, DataType(DataType.Password), Compare(nameof(Password))] public string ConfirmPassword { get; set; } = "";
        [Required, StringLength(255)] public string OrganizationName { get; set; } = "";
    }
    public void OnGet() => ReturnUrl ??= Url.Content("~/");
    public async Task<IActionResult> OnPostAsync(CancellationToken cancellationToken)
    {
        ReturnUrl ??= Url.Content("~/");
        if (!ModelState.IsValid) return Page();
        var user = new ApplicationUser { UserName = Input.Email, Email = Input.Email, EmailConfirmed = true };
        var result = await users.CreateAsync(user, Input.Password);
        if (!result.Succeeded) { foreach (var error in result.Errors) ModelState.AddModelError(string.Empty, error.Description); return Page(); }
        try
        {
            var organization = await organizations.CreateForUserAsync(user.Id, Input.OrganizationName, null, cancellationToken);
            await signInManager.SignInWithClaimsAsync(user, false, [new Claim("org_id", organization.Id.ToString())]);
            return LocalRedirect(ReturnUrl);
        }
        catch (Exception)
        {
            ModelState.AddModelError(string.Empty, "The organization could not be created. Please contact support.");
            return Page();
        }
    }
}
