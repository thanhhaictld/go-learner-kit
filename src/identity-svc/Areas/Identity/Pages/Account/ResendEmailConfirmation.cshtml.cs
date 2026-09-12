using System.ComponentModel.DataAnnotations;
using System.Text;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;
using Microsoft.AspNetCore.WebUtilities;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class ResendEmailConfirmationModel(UserManager<ApplicationUser> users, IAccountEmailSender emailSender) : PageModel
{
    [BindProperty] public InputModel Input { get; set; } = new();
    public bool Sent { get; private set; }
    public sealed class InputModel { [Required, EmailAddress] public string Email { get; set; } = ""; }
    public async Task<IActionResult> OnPostAsync(CancellationToken cancellationToken)
    {
        if (!ModelState.IsValid) return Page();
        var user = await users.FindByEmailAsync(Input.Email);
        if (user is not null && !user.EmailConfirmed)
        {
            var token = await users.GenerateEmailConfirmationTokenAsync(user);
            var code = WebEncoders.Base64UrlEncode(Encoding.UTF8.GetBytes(token));
            var link = Url.Page("ConfirmEmail", null, new { userId = user.Id, code }, Request.Scheme)!;
            await emailSender.SendAsync(user.Email!, "Confirm your email", $"<p>Confirm your account by <a href=\"{link}\">clicking this link</a>.</p>", cancellationToken);
        }
        Sent = true;
        return Page();
    }
}
