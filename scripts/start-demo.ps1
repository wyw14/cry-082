param(
    [string]$DatabaseUrl = "",
    [string]$ListenAddress = ":8080"
)

$env:HTTP_ADDR = $ListenAddress
$env:SEED_DEMO = "true"
if ($DatabaseUrl) {
    $env:DATABASE_URL = $DatabaseUrl
}

go run ./cmd/server
