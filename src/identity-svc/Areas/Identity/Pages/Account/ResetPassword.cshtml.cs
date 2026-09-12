using System.ComponentModel.DataAnnotations;
using System.Text;
using Identity.Svc.Data;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;
using Microsoft.AspNetCore.WebUtilities;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class ResetPasswordModel(UserManager<ApplicationUser> users) : PageModel
{
    [BindProperty] public InputModel Input { get; set; } = new();
    public sealed class InputModel
    {
        [Required, EmailAddress] public string Email { get; set; } = "";
        [Required, DataType(DataType.Password), StringLength(100, MinimumLength = 12)] public string Password { get; set; } = "";
        [Required, DataType(DataType.Password), Compare(nameof(Password))] public string ConfirmPassword { get; set; } = "";
        [Required] public string Code { get; set; } = "";
    }
    public IActionResult OnGet(string? code)
    {
        if (string.IsNullOrWhiteSpace(code)) return RedirectToPage("Login");
        Input.Code = code;
        return Page();
    }
    public async Task<IActionResult> OnPostAsync()
    {
        if (!ModelState.IsValid) return Page();
        var user = await users.FindByEmailAsync(Input.Email);
        if (user is null) return RedirectToPage("ResetPasswordConfirmation");
        string token;
        try { token = Encoding.UTF8.GetString(WebEncoders.Base64UrlDecode(Input.Code)); }
        catch (FormatException) { ModelState.AddModelError(string.Empty, "This reset link is invalid or has expired."); return Page(); }
        var result = await users.ResetPasswordAsync(user, token, Input.Password);
        if (result.Succeeded) return RedirectToPage("ResetPasswordConfirmation");
        foreach (var error in result.Errors) ModelState.AddModelError(string.Empty, error.Description);
        return Page();
    }
}
