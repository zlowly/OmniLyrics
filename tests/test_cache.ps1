$json = '{"title":"TestSong","artist":"TestArtist","lrc_content":"test"}'
$response = Invoke-RestMethod -Uri 'http://localhost:8080/update_cache' -Method POST -Body $json -ContentType 'application/json'
$response