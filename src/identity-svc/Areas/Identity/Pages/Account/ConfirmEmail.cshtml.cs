using System.Text;
using Identity.Svc.Data;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;
using Microsoft.AspNetCore.WebUtilities;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class ConfirmEmailModel(UserManager<ApplicationUser> users) : PageModel
{
    public string Message { get; private set; } = "Unable to confirm your email.";
    public async Task<IActionResult> OnGetAsync(string? userId, string? code)
    {
        if (string.IsNullOrWhiteSpace(userId) || string.IsNullOrWhiteSpace(code)) return Page();
        var user = await users.FindByIdAsync(userId);
        if (user is null) return Page();
        string token;
        try { token = Encoding.UTF8.GetString(WebEncoders.Base64UrlDecode(code)); }
        catch (FormatException) { return Page(); }
        var result = await users.ConfirmEmailAsync(user, token);
        Message = result.Succeeded ? "Your email has been confirmed. You can now sign in." : "This confirmation link is invalid or has expired.";
        return Page();
    }
}
