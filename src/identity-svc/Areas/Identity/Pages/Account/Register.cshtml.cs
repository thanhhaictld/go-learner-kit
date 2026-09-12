using System.ComponentModel.DataAnnotations;
using System.Security.Claims;
using System.Text;
using Microsoft.AspNetCore.WebUtilities;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class RegisterModel(
    ILogger<RegisterModel> logger,
    UserManager<ApplicationUser> users, OrganizationService organizations, IAccountEmailSender emailSender) : PageModel
{
    [BindProperty] public InputModel Input { get; set; } = new();
    [BindProperty(SupportsGet = true)] public string? ReturnUrl { get; set; }
    public sealed class InputModel
    {
        [Required, StringLength(255)] public string DisplayName { get; set; } = "";
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
        var user = new ApplicationUser { UserName = Input.Email, Email = Input.Email, DisplayName = Input.DisplayName.Trim() };
        var result = await users.CreateAsync(user, Input.Password);
        if (!result.Succeeded) { foreach (var error in result.Errors) ModelState.AddModelError(string.Empty, error.Description); return Page(); }
        try
        {
            await organizations.CreateForUserAsync(user.Id, Input.OrganizationName, null, cancellationToken);
            var token = await users.GenerateEmailConfirmationTokenAsync(user);
            var code = WebEncoders.Base64UrlEncode(Encoding.UTF8.GetBytes(token));
            var confirmationUrl = Url.Page("ConfirmEmail", null, new { userId = user.Id, code }, Request.Scheme)!;
            await emailSender.SendAsync(user.Email!, "Confirm your email", $"<p>Confirm your account by <a href=\"{confirmationUrl}\">clicking this link</a>.</p>", cancellationToken);
            return RedirectToPage("RegisterConfirmation", new { email = user.Email });
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Failed to create organization for user {UserId}", user.Id);
            ModelState.AddModelError(string.Empty, "The organization could not be created. Please contact support.");
            return Page();
        }
    }
}
