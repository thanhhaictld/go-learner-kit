#:sdk Aspire.AppHost.Sdk@13.3.5

var builder = DistributedApplication.CreateBuilder(args);

const string defaultOrganizationId = "22222222-2222-2222-2222-222222222222";
const string defaultAdminUserId = "11111111-1111-1111-1111-111111111111";
const string defaultProvisioningToken = "development-only-change-me";

var root = Path.GetFullPath("../..", Directory.GetCurrentDirectory());
var environment = ReadDevelopmentEnvironment(Path.Combine(root, ".env"));
var bootstrapOrganizationId = GetEnvironmentValue("BOOTSTRAP_ORGANIZATION_ID", defaultOrganizationId);
var bootstrapAdminUserId = GetEnvironmentValue("BOOTSTRAP_ADMIN_USER_ID", defaultAdminUserId);
var provisioningToken = GetEnvironmentValue("IDENTITY_PROVISIONING_TOKEN", defaultProvisioningToken);

var userDb = builder.AddContainer("user-db", "postgres", "17")
    .WithEnvironment("POSTGRES_USER", "app")
    .WithEnvironment("POSTGRES_PASSWORD", "app")
    .WithEnvironment("POSTGRES_DB", "users")
    .WithVolume("user-db-data", "/var/lib/postgresql/data")
    .WithEndpoint(port: 5443, targetPort: 5432, name: "tcp", scheme: "tcp", isExternal: true);

var identityDb = builder.AddContainer("identity-db", "postgres", "17")
    .WithEnvironment("POSTGRES_USER", "app")
    .WithEnvironment("POSTGRES_PASSWORD", "app")
    .WithEnvironment("POSTGRES_DB", "identity")
    .WithVolume("identity-db-data", "/var/lib/postgresql/data")
    .WithEndpoint(port: 5444, targetPort: 5432, name: "tcp", scheme: "tcp", isExternal: true);

var openFgaDb = builder.AddContainer("openfga-db", "postgres", "17")
    .WithEnvironment("POSTGRES_USER", "openfga")
    .WithEnvironment("POSTGRES_PASSWORD", "openfga")
    .WithEnvironment("POSTGRES_DB", "openfga")
    .WithVolume("openfga-db-data", "/var/lib/postgresql/data")
    .WithEndpoint(targetPort: 5432, name: "tcp", scheme: "tcp");

var userMigrate = builder.AddContainer("user-migrate", "migrate/migrate", "v4.19.1")
    .WithBindMount(Path.Combine(root, "src", "user-svc", "migrations"), "/migrations", isReadOnly: true)
    .WithArgs("-path", "/migrations", "-database", "postgres://app:app@user-db:5432/users?sslmode=disable", "up")
    .WaitFor(userDb);

var openFgaMigrate = builder.AddContainer("openfga-migrate", "openfga/openfga", "v1.10.3")
    .WithArgs("migrate")
    .WithEnvironment("OPENFGA_DATASTORE_ENGINE", "postgres")
    .WithEnvironment("OPENFGA_DATASTORE_URI", "postgres://openfga:openfga@openfga-db:5432/openfga?sslmode=disable")
    .WaitFor(openFgaDb);

var openFga = builder.AddContainer("openfga", "openfga/openfga", "v1.10.3")
    .WithArgs("run")
    .WithEnvironment("OPENFGA_DATASTORE_ENGINE", "postgres")
    .WithEnvironment("OPENFGA_DATASTORE_URI", "postgres://openfga:openfga@openfga-db:5432/openfga?sslmode=disable")
    .WithEnvironment("OPENFGA_METRICS_ADDR", "0.0.0.0:2112")
    .WithHttpEndpoint(port: 8083, targetPort: 8080, name: "http")
    .WaitForCompletion(openFgaMigrate);

var mailpit = builder.AddContainer("mailpit", "axllent/mailpit", "v1.24.1")
    .WithEndpoint(port: 1025, targetPort: 1025, name: "smtp", scheme: "smtp", isExternal: true)
    .WithHttpEndpoint(port: 8025, targetPort: 8025, name: "http");

var authzService = builder.AddExecutable(
        "authz-service",
        "go",
        Path.Combine(root, "src", "authz-svc"),
        "tool", "air", "-c", ".air.toml")
    .WithEnvironment("HTTP_PORT", "8082")
    .WithEnvironment("FGA_API_URL", "http://localhost:8083")
    .WithEnvironment("FGA_STORE_NAME", "go-learner-authz")
    .WithEnvironment("BOOTSTRAP_ORGANIZATION_ID", bootstrapOrganizationId)
    .WithEnvironment("BOOTSTRAP_ADMIN_USER_ID", bootstrapAdminUserId)
    .WithEnvironment("IDENTITY_PROVISIONING_TOKEN", provisioningToken)
    .WithHttpEndpoint(port: 8082, name: "http")
    .WithHttpHealthCheck("/health/ready")
    .WaitFor(openFga);

var identityMigrate = builder.AddExecutable(
        "identity-migrate",
        "dotnet",
        Path.Combine(root, "src", "identity-svc"),
        "run", "--", "--migrate")
    .WithEnvironment("ASPNETCORE_ENVIRONMENT", "Development")
    .WithEnvironment("ConnectionStrings__Default", "Host=localhost;Port=5444;Database=identity;Username=app;Password=app")
    .WithEnvironment("OpenIddict__Issuer", "http://localhost:8081")
    .WithEnvironment("OpenIddict__AllowInsecureHttp", "true")
    .WithEnvironment("Bootstrap__OrganizationId", bootstrapOrganizationId)
    .WithEnvironment("Bootstrap__AdminUserId", bootstrapAdminUserId)
    .WithEnvironment("Bootstrap__AdminEmail", GetEnvironmentValue("BOOTSTRAP_ADMIN_EMAIL", "admin@example.test"))
    .WithEnvironment("Bootstrap__AdminPassword", GetEnvironmentValue("BOOTSTRAP_ADMIN_PASSWORD", "changeMe@123!"))
    .WaitFor(identityDb);

var userService = builder.AddExecutable(
        "user-service",
        "go",
        Path.Combine(root, "src", "user-svc"),
        "tool", "air", "-c", ".air.toml")
    .WithEnvironment("HTTP_PORT", "8080")
    .WithEnvironment("DB_HOST", "localhost")
    .WithEnvironment("DB_PORT", "5443")
    .WithEnvironment("DB_USER", "app")
    .WithEnvironment("DB_PASSWORD", "app")
    .WithEnvironment("DB_NAME", "users")
    .WithEnvironment("DB_SSLMODE", "disable")
    .WithEnvironment("AUTHZ_SERVICE_URL", "http://localhost:8082")
    .WithHttpEndpoint(port: 8080, name: "http")
    .WithHttpHealthCheck("/health/ready")
    .WaitForCompletion(userMigrate)
    .WaitFor(authzService);

var identityService = builder.AddExecutable(
        "identity-service",
        "dotnet",
        Path.Combine(root, "src", "identity-svc"),
        "watch", "run", "--no-hot-reload", "--urls", "http://localhost:8081")
    .WithEnvironment("ASPNETCORE_ENVIRONMENT", "Development")
    .WithEnvironment("ConnectionStrings__Default", "Host=localhost;Port=5444;Database=identity;Username=app;Password=app")
    .WithEnvironment("Authz__ServiceUrl", "http://localhost:8082")
    .WithEnvironment("Authz__ProvisioningToken", provisioningToken)
    .WithEnvironment("Email__Smtp__Host", "localhost")
    .WithEnvironment("Email__Smtp__Port", "1025")
    .WithEnvironment("Email__Smtp__EnableSsl", "false")
    .WithHttpEndpoint(port: 8081, name: "http")
    .WithHttpHealthCheck("/health/ready")
    .WaitForCompletion(identityMigrate)
    .WaitFor(authzService)
    .WaitFor(mailpit);

var vite = builder.AddExecutable(
        "saas-admin-vite",
        "npm",
        Path.Combine(root, "apps", "saas-admin-portal"),
        "run", "dev")
    .WithHttpEndpoint(port: 5173, name: "http");

builder.AddExecutable(
        "saas-admin-bff",
        "dotnet",
        Path.Combine(root, "apps", "saas-admin-portal", "bff-saas-app"),
        "watch", "run", "--no-hot-reload", "--urls", "http://localhost:3001")
    .WithEnvironment("ASPNETCORE_ENVIRONMENT", "Development")
    .WithEnvironment("Authentication__BackchannelAuthority", "http://localhost:8081")
    .WithEnvironment("ReverseProxy__Clusters__user-service__Destinations__primary__Address", "http://localhost:8080/")
    .WithEnvironment("ReverseProxy__Clusters__vite__Destinations__primary__Address", "http://127.0.0.1:5173/")
    .WithHttpEndpoint(port: 3001, name: "http")
    .WithHttpHealthCheck("/bff/ready")
    .WaitFor(identityService)
    .WaitFor(userService)
    .WaitFor(vite);

var prometheus = builder.AddContainer("prometheus", "prom/prometheus", "v3.5.0")
    .WithArgs("--config.file=/etc/prometheus/prometheus.watch.yml", "--storage.tsdb.path=/prometheus", "--web.enable-remote-write-receiver")
    .WithBindMount(Path.Combine(root, "platform", "monitoring", "prometheus", "prometheus.watch.yml"), "/etc/prometheus/prometheus.watch.yml", isReadOnly: true)
    .WithVolume("prometheus-data", "/prometheus")
    .WithHttpEndpoint(port: 9090, targetPort: 9090, name: "http")
    .WaitFor(userService)
    .WaitFor(authzService)
    .WaitFor(identityService)
    .WaitFor(openFga);

builder.AddContainer("grafana", "grafana/grafana", "12.1.0")
    .WithEnvironment("GF_SECURITY_ADMIN_USER", "admin")
    .WithEnvironment("GF_SECURITY_ADMIN_PASSWORD", "admin")
    .WithEnvironment("GF_AUTH_ANONYMOUS_ENABLED", "true")
    .WithEnvironment("GF_AUTH_ANONYMOUS_ORG_ROLE", "Viewer")
    .WithBindMount(Path.Combine(root, "platform", "monitoring", "grafana", "provisioning"), "/etc/grafana/provisioning", isReadOnly: true)
    .WithBindMount(Path.Combine(root, "platform", "monitoring", "grafana", "dashboards"), "/var/lib/grafana/dashboards", isReadOnly: true)
    .WithVolume("grafana-data", "/var/lib/grafana")
    .WithHttpEndpoint(port: 3000, targetPort: 3000, name: "http")
    .WaitFor(prometheus);

builder.Build().Run();

string GetEnvironmentValue(string name, string fallback) =>
    Environment.GetEnvironmentVariable(name) ??
    (environment.TryGetValue(name, out var value) ? value : fallback);

static Dictionary<string, string> ReadDevelopmentEnvironment(string path)
{
    var values = new Dictionary<string, string>(StringComparer.Ordinal);
    if (!File.Exists(path)) return values;

    foreach (var line in File.ReadLines(path))
    {
        var separator = line.IndexOf('=');
        if (separator <= 0 || line.TrimStart().StartsWith('#')) continue;
        values[line[..separator].Trim()] = line[(separator + 1)..].Trim();
    }

    return values;
}
