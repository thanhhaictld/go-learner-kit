using Microsoft.AspNetCore.Identity;

namespace Identity.Svc.Data;

public sealed class ApplicationUser : IdentityUser
{
    public string DisplayName { get; set; } = "";
    public DateTimeOffset CreatedAt { get; set; } = DateTimeOffset.UtcNow;
    public ICollection<OrganizationMembership> Memberships { get; set; } = new List<OrganizationMembership>();
}
