param(
    [ValidateSet("windows","linux","macos")] [string]$Platform = "windows",
    [int]$Iterations = 3
)

$benchFiles = @(
    @{ Name = "string_concat"; Path = "bench\\string_concat.bz" },
    @{ Name = "string_builder"; Path = "bench\\string_builder.bz" },
    @{ Name = "json_validate"; Path = "bench\\json_validate.bz" },
    @{ Name = "crypto_sha256"; Path = "bench\\crypto_sha256.bz" },
    @{ Name = "parse_int_float"; Path = "bench\\parse_int_float.bz" },
    @{ Name = "loop_arith"; Path = "bench\\loop_arith.bz" },
    @{ Name = "match_hot"; Path = "bench\\match_hot.bz" },
    @{ Name = "base64_roundtrip"; Path = "bench\\base64_roundtrip.bz" },
    @{ Name = "jwt_sign_verify"; Path = "bench\\jwt_sign_verify.bz" }
)

function Run-Bench($backend, $name, $path, $iters) {
    $best = -1
    for ($i = 0; $i -lt $iters; $i++) {
        $output = & go run .\cmd\bazc\ run $path --backend $backend 2>$null
        $timeMs = $output | Select-Object -First 1
        if ($timeMs -match '^[0-9]+$') {
            $val = [int]$timeMs
            if ($best -lt 0 -or $val -lt $best) { $best = $val }
        }
    }
    return $best
}

$go = @{}
$llvm = @{}
foreach ($b in $benchFiles) {
    $go[$b.Name] = Run-Bench "go" $b.Name $b.Path $Iterations
    $llvm[$b.Name] = Run-Bench "llvm" $b.Name $b.Path $Iterations
}

$doc = New-Object System.Xml.XmlDocument
$root = $doc.CreateElement("benchmarks")
$baseline = $doc.CreateElement("baseline")
$baseline.SetAttribute("platform", $Platform)
$baseline.SetAttribute("threshold_percent", "40")

$goNode = $doc.CreateElement("go")
foreach ($k in $go.Keys) {
    $n = $doc.CreateElement($k)
    $n.InnerText = [string]$go[$k]
    $goNode.AppendChild($n) | Out-Null
}

$llvmNode = $doc.CreateElement("llvm")
foreach ($k in $llvm.Keys) {
    $n = $doc.CreateElement($k)
    $n.InnerText = [string]$llvm[$k]
    $llvmNode.AppendChild($n) | Out-Null
}

$baseline.AppendChild($goNode) | Out-Null
$baseline.AppendChild($llvmNode) | Out-Null
$root.AppendChild($baseline) | Out-Null
$doc.AppendChild($root) | Out-Null

$doc.Save("bench\\baseline.xml")
Write-Host "Saved bench\\baseline.xml"
