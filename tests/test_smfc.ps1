Add-Type -AssemblyName System.Runtime.WindowsRuntime -ErrorAction SilentlyContinue
$mgrType = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows, ContentType=WindowsRuntime]
if ($null -eq $mgrType) { Write-Output "NO_TYPE"; exit }
$mgr = $null
try { $mgr = $mgrType::GetDefault() } catch { $mgr = $null }
if ($null -eq $mgr) { try { $a = $mgrType::GetDefaultAsync(); $mgr = $a.GetResults() } catch { $mgr = $null } }
if ($null -eq $mgr) { Write-Output "NO_MANAGER"; exit }
$session = $mgr.GetCurrentSession()
if ($null -eq $session) { Write-Output "NO_SESSION"; exit }
Write-Output "HAS_SESSION"