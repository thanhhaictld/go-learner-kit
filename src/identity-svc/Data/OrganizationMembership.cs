namespace Identity.Svc.Data;

public sealed class OrganizationMembership
{
    public Guid OrganizationId { get; set; }
    public required string UserId { get; set; }
    public DateTimeOffset JoinedAt { get; set; } = DateTimeOffset.UtcNow;
    public Organization Organization { get; set; } = null!;
    public ApplicationUser User { get; set; } = null!;
}
