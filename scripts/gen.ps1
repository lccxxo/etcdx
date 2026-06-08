# scripts/gen.ps1
# 把 api/**/*.proto 编译成 Go 代码（含 gRPC stub）
#
# 用法：
#   从项目任意目录运行：powershell -File scripts\gen.ps1
#   或者：cd 到项目根目录，然后：.\scripts\gen.ps1

$ErrorActionPreference = "Stop"

# --- 1. 切换到项目根（脚本所在目录的上一层） ---
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot
Write-Host "[gen] project root: $ProjectRoot"

# --- 2. 检查工具链 ---
$required = @("protoc", "protoc-gen-go", "protoc-gen-go-grpc")
$missing  = @()
foreach ($tool in $required) {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        $missing += $tool
    }
}
if ($missing.Count -gt 0) {
    Write-Host "[gen] missing tool(s): $($missing -join ', ')" -ForegroundColor Red
    Write-Host "[gen] install hints:"
    Write-Host "      protoc           -> https://github.com/protocolbuffers/protobuf/releases"
    Write-Host "      protoc-gen-go    -> go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    Write-Host "      protoc-gen-grpc  -> go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    Write-Host "      (确保 %USERPROFILE%\go\bin 已加入 PATH)"
    exit 1
}

# --- 3. 收集所有 .proto 文件 ---
$protos = Get-ChildItem -Path "api" -Filter "*.proto" -Recurse -ErrorAction SilentlyContinue
if ($protos.Count -eq 0) {
    Write-Host "[gen] no .proto files found under api/" -ForegroundColor Yellow
    exit 0
}

# 转换为相对于项目根的、用正斜杠的路径
# （Windows 下 protoc 对 ".\xxx" 和反斜杠路径敏感，会把 "." 误判成一个输入目录）
$relPaths = @()
foreach ($p in $protos) {
    $rel = $p.FullName.Substring($ProjectRoot.Length + 1).Replace('\', '/')
    $relPaths += $rel
}

Write-Host "[gen] found $($protos.Count) proto file(s):"
foreach ($rel in $relPaths) {
    Write-Host "       $rel"
}

# --- 4. 调 protoc 一次性生成 ---
# --proto_path=. 显式锁定 import 解析根为项目根，避免 protoc 自己瞎猜
& protoc `
    --proto_path=. `
    --go_out=. --go_opt=paths=source_relative `
    --go-grpc_out=. --go-grpc_opt=paths=source_relative `
    @relPaths

if ($LASTEXITCODE -ne 0) {
    Write-Host "[gen] protoc exited with code $LASTEXITCODE" -ForegroundColor Red
    exit $LASTEXITCODE
}

Write-Host "[gen] done." -ForegroundColor Green
