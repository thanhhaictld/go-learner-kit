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

        var clients = configuration.GetSection("OpenIddict:Applications").Get<List<OpenIddictClientSettings>>() ?? [];
        foreach (var client in clients)
        {
            if (string.IsNullOrWhiteSpace(client.ClientId)) throw new InvalidOperationException("OpenIddict clientId is required.");
            var descriptor = CreateDescriptor(client);
            var existing = await applications.FindByClientIdAsync(client.ClientId, cancellationToken);
            if (existing is null)
                await applications.CreateAsync(descriptor, cancellationToken);
            else
                await applications.UpdateAsync(existing, descriptor, cancellationToken);
        }
    }

    private static OpenIddictApplicationDescriptor CreateDescriptor(OpenIddictClientSettings client)
    {
        var descriptor = new OpenIddictApplicationDescriptor
        {
            ClientId = client.ClientId,
            DisplayName = client.DisplayName,
            ClientType = client.ClientType,
            ConsentType = OpenIddictConstants.ConsentTypes.Implicit
        };
        foreach (var uri in client.RedirectUris) descriptor.RedirectUris.Add(new Uri(uri));
        foreach (var uri in client.PostLogoutRedirectUris) descriptor.PostLogoutRedirectUris.Add(new Uri(uri));
        descriptor.Permissions.UnionWith([
            OpenIddictConstants.Permissions.Endpoints.Authorization,
            OpenIddictConstants.Permissions.Endpoints.Token,
            OpenIddictConstants.Permissions.Endpoints.Logout,
            OpenIddictConstants.Permissions.GrantTypes.AuthorizationCode,
            OpenIddictConstants.Permissions.GrantTypes.RefreshToken,
            OpenIddictConstants.Permissions.ResponseTypes.Code,
            OpenIddictConstants.Permissions.Prefixes.Scope + OpenIddictConstants.Scopes.OpenId,
            OpenIddictConstants.Permissions.Prefixes.Scope + OpenIddictConstants.Scopes.Email,
            OpenIddictConstants.Permissions.Prefixes.Scope + OpenIddictConstants.Scopes.Profile,
            OpenIddictConstants.Permissions.Prefixes.Scope + OpenIddictConstants.Scopes.OfflineAccess
        ]);
        return descriptor;
    }
}

public sealed class OpenIddictClientSettings
{
    public string ClientId { get; set; } = "";
    public string DisplayName { get; set; } = "";
    public string ClientType { get; set; } = OpenIddictConstants.ClientTypes.Public;
    public List<string> RedirectUris { get; set; } = [];
    public List<string> PostLogoutRedirectUris { get; set; } = [];
}
