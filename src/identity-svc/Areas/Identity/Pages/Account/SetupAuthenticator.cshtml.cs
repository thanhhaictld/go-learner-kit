using System.Security.Claims;
using System.Text.Encodings.Web;
using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.RazorPages;
using QRCoder;

namespace Identity.Svc.Areas.Identity.Pages.Account;

public sealed class SetupAuthenticatorModel(UserManager<ApplicationUser> users, SignInManager<ApplicationUser> signInManager, IConfiguration configuration) : PageModel
{
    [BindProperty] public string Code { get; set; } = "";
    public string SharedKey { get; private set; } = "";
    public string AuthenticatorUri { get; private set; } = "";
    public string? QrCodeDataUrl { get; private set; }
    public IReadOnlyList<string> RecoveryCodes { get; private set; } = [];

    public async Task<IActionResult> OnGetAsync()
    {
        var user = await ResolveUserAsync();
        if (user is null) return RedirectToPage("Login");
        await LoadAsync(user);
        return Page();
    }

    public async Task<IActionResult> OnPostAsync()
    {
        var user = await ResolveUserAsync();
        var context = await ResolveContextAsync();
        if (user is null || context.OrganizationId is null) return RedirectToPage("Login");
        var code = Code.Replace(" ", "", StringComparison.Ordinal).Replace("-", "", StringComparison.Ordinal);
        if (!await users.VerifyTwoFactorTokenAsync(user, users.Options.Tokens.AuthenticatorTokenProvider, code))
        {
            ModelState.AddModelError(string.Empty, "Verification code is invalid.");
            await LoadAsync(user);
            return Page();
        }
        await users.SetTwoFactorEnabledAsync(user, true);
        RecoveryCodes = (await users.GenerateNewTwoFactorRecoveryCodesAsync(user, 10) ?? []).ToArray();
        if (context.Pending)
        {
            await PendingSignIn.ClearAsync(HttpContext);
            await signInManager.SignInWithClaimsAsync(user, context.RememberMe, [new Claim("org_id", context.OrganizationId.Value.ToString())]);
        }
        else await signInManager.SignInWithClaimsAsync(user, false, [new Claim("org_id", context.OrganizationId.Value.ToString())]);
        return Page();
    }

    private async Task LoadAsync(ApplicationUser user)
    {
        var key = await users.GetAuthenticatorKeyAsync(user);
        if (string.IsNullOrEmpty(key)) { await users.ResetAuthenticatorKeyAsync(user); key = await users.GetAuthenticatorKeyAsync(user); }
        SharedKey = FormatKey(key!);
        var issuer = configuration["Mfa:Issuer"] ?? "SaaS Identity";
        AuthenticatorUri = $"otpauth://totp/{Uri.EscapeDataString(issuer)}:{Uri.EscapeDataString(user.Email!)}?secret={key}&issuer={Uri.EscapeDataString(issuer)}&digits=6";
        using var generator = new QRCodeGenerator();
        using var data = generator.CreateQrCode(AuthenticatorUri, QRCodeGenerator.ECCLevel.Q);
        QrCodeDataUrl = "data:image/png;base64," + Convert.ToBase64String(new PngByteQRCode(data).GetGraphic(20));
    }

    private async Task<ApplicationUser?> ResolveUserAsync()
    {
        var current = await users.GetUserAsync(User);
        if (current is not null) return current;
        var pending = await PendingSignIn.GetAsync(HttpContext);
        return pending is null ? null : await users.FindByIdAsync(pending.UserId);
    }

    private async Task<(Guid? OrganizationId, bool Pending, bool RememberMe)> ResolveContextAsync()
    {
        var pending = await PendingSignIn.GetAsync(HttpContext);
        if (pending is not null) return (pending.OrganizationId, true, pending.RememberMe);
        return (Guid.TryParse(User.FindFirstValue("org_id"), out var organizationId) ? organizationId : null, false, false);
    }

    private static string FormatKey(string key) => string.Join(" ", Enumerable.Range(0, (key.Length + 3) / 4).Select(index => key.Substring(index * 4, Math.Min(4, key.Length - index * 4))));
}
