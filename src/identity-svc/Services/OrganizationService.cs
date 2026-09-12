using Identity.Svc.Data;
using Microsoft.EntityFrameworkCore;

namespace Identity.Svc.Services;

public sealed class OrganizationService(ApplicationDbContext db, OpenFgaProvisioner provisioner)
{
    public Task<List<Organization>> GetForUserAsync(string userId, CancellationToken cancellationToken) =>
        db.OrganizationMemberships
            .Where(x => x.UserId == userId)
            .OrderBy(x => x.Organization.Name)
            .Select(x => x.Organization)
            .ToListAsync(cancellationToken);

    public async Task<Organization> CreateForUserAsync(string userId, string name, string? description, CancellationToken cancellationToken)
    {
        var normalizedName = name.Trim();
        if (string.IsNullOrWhiteSpace(normalizedName)) throw new ArgumentException("Organization name is required.");
        var slug = CreateSlug(normalizedName);
        if (await db.Organizations.AnyAsync(x => x.Slug == slug, cancellationToken))
            throw new InvalidOperationException("An organization with this name already exists.");

        var organization = new Organization { Name = normalizedName, Slug = slug, Description = description?.Trim() };
        db.Organizations.Add(organization);
        db.OrganizationMemberships.Add(new OrganizationMembership { OrganizationId = organization.Id, UserId = userId });
        await db.SaveChangesAsync(cancellationToken);
        await provisioner.AssignOrganizationAdminAsync(organization.Id, userId, cancellationToken);
        return organization;
    }

    private static string CreateSlug(string name)
    {
        var slug = string.Concat(name.ToLowerInvariant().Select(c => char.IsLetterOrDigit(c) ? c : '-')).Trim('-');
        return string.IsNullOrEmpty(slug) ? "organization" : slug[..Math.Min(100, slug.Length)];
    }
}
