using System.ComponentModel.DataAnnotations;
using System.Text;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;
using Microsoft.AspNetCore.WebUtilities;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class ForgotPasswordModel(UserManager<ApplicationUser> users, IAccountEmailSender emailSender) : PageModel
{
    [BindProperty] public InputModel Input { get; set; } = new();
    public sealed class InputModel { [Required, EmailAddress] public string Email { get; set; } = ""; }
    public async Task<IActionResult> OnPostAsync(CancellationToken cancellationToken)
    {
        if (!ModelState.IsValid) return Page();
        var user = await users.FindByEmailAsync(Input.Email);
        if (user is not null && user.EmailConfirmed)
        {
            var token = await users.GeneratePasswordResetTokenAsync(user);
            var code = WebEncoders.Base64UrlEncode(Encoding.UTF8.GetBytes(token));
            var link = Url.Page("ResetPassword", null, new { code }, Request.Scheme)!;
            await emailSender.SendAsync(user.Email!, "Reset your password", $"<p>Reset your password by <a href=\"{link}\">clicking this link</a>.</p>", cancellationToken);
        }
        return RedirectToPage("ForgotPasswordConfirmation");
    }
}
