
namespace Identity.Svc.Configurations;

public class OpenFgaConfig
{
    public string ServiceUrl { get; set; } = "";
    public string ProvisioningToken { get; set; } = "";
    public bool SkipChecksInDebug { get; set; } = true;
}
