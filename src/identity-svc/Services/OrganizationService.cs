using Identity.Svc.Data;
using Microsoft.EntityFrameworkCore;

namespace Identity.Svc.Services;

public sealed class OrganizationService(ApplicationDbContext db, OpenFgaProvisioner provisioner, ILogger<OrganizationService> logger)
{
    public Task<List<Organization>> GetForUserAsync(string userId, CancellationToken cancellationToken) =>
        db.OrganizationMemberships
            .Where(x => x.UserId == userId)
            .OrderBy(x => x.Organization.Name)
            .Select(x => x.Organization)
            .ToListAsync(cancellationToken);

    public Task<List<OrganizationMembership>> GetMembershipsForUserAsync(string userId, CancellationToken cancellationToken) =>
        db.OrganizationMemberships.Include(x => x.Organization).Where(x => x.UserId == userId)
            .OrderBy(x => x.Organization.Name).ToListAsync(cancellationToken);

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
        _ = Task.Run(async () =>
        {
            try
            {
                await provisioner.AssignOrganizationAdminAsync(organization.Id, userId, cancellationToken);
            }
            catch (Exception ex)
            {
                // Log the error but do not block the user creation process
                logger.LogError(ex, "Failed to assign admin role for organization {OrganizationId}", organization.Id);
            }
        }, cancellationToken);
        return organization;
    }

    public Task<Organization?> GetForUserAsync(Guid organizationId, string userId, CancellationToken cancellationToken) =>
        db.OrganizationMemberships.Where(x => x.OrganizationId == organizationId && x.UserId == userId)
            .Select(x => x.Organization).SingleOrDefaultAsync(cancellationToken);

    public async Task<bool> UpdateMfaPolicyAsync(Guid organizationId, string userId, bool required, CancellationToken cancellationToken)
    {
        if (!await provisioner.CanManageOrganizationAsync(organizationId, userId, cancellationToken)) return false;
        var organization = await db.Organizations.FindAsync([organizationId], cancellationToken);
        if (organization is null) return false;
        organization.MfaRequired = required;
        await db.SaveChangesAsync(cancellationToken);
        return true;
    }

    private static string CreateSlug(string name)
    {
        var slug = string.Concat(name.ToLowerInvariant().Select(c => char.IsLetterOrDigit(c) ? c : '-')).Trim('-');
        return string.IsNullOrEmpty(slug) ? "organization" : slug[..Math.Min(100, slug.Length)];
    }
}
