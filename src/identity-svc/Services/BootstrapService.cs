using Identity.Svc.Data;
using Microsoft.AspNetCore.Identity;
using Microsoft.EntityFrameworkCore;
using OpenIddict.Abstractions;

namespace Identity.Svc.Services;

public sealed class BootstrapService(ApplicationDbContext db, UserManager<ApplicationUser> users, IOpenIddictApplicationManager applications, IConfiguration configuration)
{
    public async Task MigrateAndSeedAsync(CancellationToken cancellationToken)
    {
        await db.Database.MigrateAsync(cancellationToken);
        var organizationId = Guid.Parse(configuration["Bootstrap:OrganizationId"] ?? "22222222-2222-2222-2222-222222222222");
        var userId = configuration["Bootstrap:AdminUserId"] ?? "11111111-1111-1111-1111-111111111111";
        var email = configuration["Bootstrap:AdminEmail"] ?? "admin@example.test";
        var password = configuration["Bootstrap:AdminPassword"] ?? "changeMe@123!";

        var organization = await db.Organizations.FindAsync([organizationId], cancellationToken);
        if (organization is null)
        {
            organization = new Organization { Id = organizationId, Name = "Bootstrap Organization", Slug = "bootstrap" };
            db.Organizations.Add(organization);
        }
        var user = await users.FindByIdAsync(userId);
        if (user is null)
        {
            user = new ApplicationUser { Id = userId, UserName = email, Email = email, DisplayName = email, EmailConfirmed = true };
            var result = await users.CreateAsync(user, password);
            if (!result.Succeeded) throw new InvalidOperationException(string.Join("; ", result.Errors.Select(x => x.Description)));
        }
        if (!await db.OrganizationMemberships.AnyAsync(x => x.OrganizationId == organizationId && x.UserId == userId, cancellationToken))
            db.OrganizationMemberships.Add(new OrganizationMembership { OrganizationId = organizationId, UserId = userId });
        await db.SaveChangesAsync(cancellationToken);

        if (await applications.FindByClientIdAsync("saas-web", cancellationToken) is null)
        {
            var redirectUri = configuration["Bootstrap:ClientRedirectUri"] ?? "http://localhost:3001/auth/callback";
            await applications.CreateAsync(new OpenIddictApplicationDescriptor
            {
                ClientId = "saas-web",
                DisplayName = "SaaS Web",
                ConsentType = OpenIddictConstants.ConsentTypes.Implicit,
                RedirectUris = { new Uri(redirectUri) },
                Permissions =
                {
                    OpenIddictConstants.Permissions.Endpoints.Authorization,
                    OpenIddictConstants.Permissions.Endpoints.Token,
                    OpenIddictConstants.Permissions.GrantTypes.AuthorizationCode,
                    OpenIddictConstants.Permissions.GrantTypes.RefreshToken,
                    OpenIddictConstants.Permissions.ResponseTypes.Code,
                    OpenIddictConstants.Permissions.Prefixes.Scope + OpenIddictConstants.Scopes.OpenId,
                    OpenIddictConstants.Permissions.Prefixes.Scope + OpenIddictConstants.Scopes.Email,
                    OpenIddictConstants.Permissions.Prefixes.Scope + OpenIddictConstants.Scopes.Profile,
                    OpenIddictConstants.Permissions.Prefixes.Scope + OpenIddictConstants.Scopes.OfflineAccess
                }
            }, cancellationToken);
        }
    }
}
