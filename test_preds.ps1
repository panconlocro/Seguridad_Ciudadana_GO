$loginData = @{
    usuario = "admin"
    password = "admin123"
} | ConvertTo-Json

try {
    $loginRes = Invoke-RestMethod -Uri "http://localhost:8080/login" -Method Post -Body $loginData -ContentType "application/json"
    $token = $loginRes.datos.token
} catch {
    Write-Host "Error en login: $_"
    exit
}

$headers = @{
    Authorization = "Bearer $token"
}

Write-Host "Enviando 50 predicciones ALEATORIAS para generar carga en los nodos..."

for ($i = 1; $i -le 50; $i++) {
    $r = Get-Random -Minimum 1 -Maximum 4
    
    # Generar datos aleatorios cada vez para evadir el cache de Redis
    $randHour = Get-Random -Minimum 0 -Maximum 24
    $randDay = Get-Random -Minimum 0 -Maximum 7
    $randArea = Get-Random -Minimum 1 -Maximum 22
    $randPremis = Get-Random -Minimum 100 -Maximum 500
    
    try {
        if ($r -eq 1) {
            $data = @{ hour = $randHour; day_of_week = $randDay; month = 6; area = $randArea; premis_cd = $randPremis; part_1_2 = 1; victim_identified = 0; days_to_report = 0 } | ConvertTo-Json
            Invoke-RestMethod -Uri "http://localhost:8080/predict/crime-type" -Method Post -Body $data -ContentType "application/json" -Headers $headers | Out-Null
            Write-Host "[$i/50] -> Model 1 (MISS forzado)"
        } elseif ($r -eq 2) {
            $data = @{ hour = $randHour; day_of_week = $randDay; month = 6; crm_cd = 200; premis_cd = $randPremis; part_1_2 = 1; area = $randArea } | ConvertTo-Json
            Invoke-RestMethod -Uri "http://localhost:8080/predict/risk-zone" -Method Post -Body $data -ContentType "application/json" -Headers $headers | Out-Null
            Write-Host "[$i/50] -> Model 2 (MISS forzado)"
        } else {
            $data = @{ crm_cd = 200; area = $randArea; hour = $randHour; day_of_week = $randDay; premis_cd = $randPremis; weapon_present = 0; victim_identified = 0; days_to_report = 0; part_1_2 = 1 } | ConvertTo-Json
            Invoke-RestMethod -Uri "http://localhost:8080/predict/arrest-prob" -Method Post -Body $data -ContentType "application/json" -Headers $headers | Out-Null
            Write-Host "[$i/50] -> Model 3 (MISS forzado)"
        }
    } catch {
        Write-Host "Error en request $i : $_"
    }
    
    Start-Sleep -Milliseconds 250
}
Write-Host "¡Listo!"
