$loginData = @{
    usuario = "admin"
    password = "admin123"
} | ConvertTo-Json

try {
    $loginRes = Invoke-RestMethod -Uri "http://localhost:8080/login" -Method Post -Body $loginData -ContentType "application/json"
    $token = $loginRes.datos.token
} catch {
    Write-Host "Error en login"
    exit
}

$headers = @{
    Authorization = "Bearer $token"
}

$model1Data = @{ hour = 12; day_of_week = 3; month = 6; area = 1; premis_cd = 101; part_1_2 = 1; victim_identified = 0; days_to_report = 0 } | ConvertTo-Json
$model2Data = @{ hour = 12; day_of_week = 3; month = 6; crm_cd = 200; premis_cd = 101; part_1_2 = 1; area = 1 } | ConvertTo-Json
$model3Data = @{ crm_cd = 200; area = 1; hour = 12; day_of_week = 3; premis_cd = 101; weapon_present = 0; victim_identified = 0; days_to_report = 0; part_1_2 = 1 } | ConvertTo-Json

Write-Host "Enviando 30 predicciones aleatorias..."

for ($i = 1; $i -le 30; $i++) {
    $r = Get-Random -Minimum 1 -Maximum 4
    try {
        if ($r -eq 1) {
            Invoke-RestMethod -Uri "http://localhost:8080/predict/crime-type" -Method Post -Body $model1Data -ContentType "application/json" -Headers $headers | Out-Null
            Write-Host "[$i/30] -> Model 1 (Tipo de Crimen)"
        } elseif ($r -eq 2) {
            Invoke-RestMethod -Uri "http://localhost:8080/predict/risk-zone" -Method Post -Body $model2Data -ContentType "application/json" -Headers $headers | Out-Null
            Write-Host "[$i/30] -> Model 2 (Zona de Riesgo)"
        } else {
            Invoke-RestMethod -Uri "http://localhost:8080/predict/arrest-prob" -Method Post -Body $model3Data -ContentType "application/json" -Headers $headers | Out-Null
            Write-Host "[$i/30] -> Model 3 (Prob. Arresto)"
        }
    } catch {
        Write-Host "Error en request $i : $_"
    }
    
    # Pausa de 300ms entre requests para que se vea bien en el frontend
    Start-Sleep -Milliseconds 300
}

Write-Host "¡Listo!"
