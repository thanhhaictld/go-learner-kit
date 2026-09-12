using Identity.Svc.Data;
using Identity.Svc.Services;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;

namespace Identity.Svc.Controllers;

[ApiController]
[Authorize]
[Route("api/organizations")]
public sealed class OrganizationController(UserManager<ApplicationUser> users, OrganizationService organizations) : ControllerBase
{
    [HttpGet("mine")]
    public async Task<IActionResult> Mine(CancellationToken cancellationToken)
    {
        var user = await users.GetUserAsync(User);
        if (user is null) return Unauthorized();
        var result = await organizations.GetForUserAsync(user.Id, cancellationToken);
        return Ok(result.Select(x => new { x.Id, x.Name, x.Slug, x.Description, x.MfaRequired }));
    }

    [HttpPost]
    public async Task<IActionResult> Create(CreateOrganizationRequest request, CancellationToken cancellationToken)
    {
        var user = await users.GetUserAsync(User);
        if (user is null) return Unauthorized();
        var organization = await organizations.CreateForUserAsync(user.Id, request.Name, request.Description, cancellationToken);
        return Created($"/api/organizations/{organization.Id}", new { organization.Id, organization.Name, organization.Slug });
    }
}

public sealed record CreateOrganizationRequest(string Name, string? Description);
