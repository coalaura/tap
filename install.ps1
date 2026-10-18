& {
	$ErrorActionPreference = 'Stop'
	$ProgressPreference = 'SilentlyContinue'

	if ($env:OS -ne 'Windows_NT') {
		throw 'This installer requires Windows.'
	}

	$architecture = $env:PROCESSOR_ARCHITEW6432
	if (-not $architecture) {
		$architecture = $env:PROCESSOR_ARCHITECTURE
	}

	if ($architecture -ne 'AMD64') {
		throw 'Only Windows amd64 (x86_64) is supported.'
	}

	if (-not $env:USERPROFILE) {
		throw 'USERPROFILE is not set.'
	}

	$directory = Join-Path $env:USERPROFILE '.bin'
	$destination = Join-Path $directory 'tap.exe'
	$staging = Join-Path $directory ('.tap-install-' + [Guid]::NewGuid().ToString('N'))
	$securityProtocol = [Net.ServicePointManager]::SecurityProtocol

	try {
		[Net.ServicePointManager]::SecurityProtocol = $securityProtocol -bor [Net.SecurityProtocolType]::Tls12
		$headers = @{ 'User-Agent' = 'tap-installer'; 'Accept' = 'application/vnd.github+json' }
		if ($env:GITHUB_TOKEN) {
			$headers['Authorization'] = 'Bearer ' + $env:GITHUB_TOKEN
		}

		$release = Invoke-RestMethod -Uri 'https://api.github.com/repos/coalaura/tap/releases/latest' -Headers $headers
		if (-not $release.tag_name -or $release.draft -or $release.prerelease) {
			throw 'Could not resolve the latest stable release.'
		}

		$tag = [Uri]::EscapeDataString($release.tag_name)
		$asset = 'tap-windows-amd64.exe'
		$releaseUrl = "https://github.com/coalaura/tap/releases/download/$tag"
		$null = [IO.Directory]::CreateDirectory($staging)
		$download = Join-Path $staging $asset
		$checksums = Join-Path $staging 'SHA256SUMS'

		Write-Host "Downloading tap $($release.tag_name)..."
		Invoke-WebRequest -UseBasicParsing -Uri "$releaseUrl/$asset" -OutFile $download
		Invoke-WebRequest -UseBasicParsing -Uri "$releaseUrl/SHA256SUMS" -OutFile $checksums

		$checksum = $null
		foreach ($line in [IO.File]::ReadAllLines($checksums)) {
			if ($line -match '^([a-fA-F0-9]{64}) [ *]tap-windows-amd64\.exe$') {
				if ($checksum) {
					throw 'Duplicate release checksum.'
				}

				$checksum = $Matches[1]
			}
		}

		if (-not $checksum) {
			throw 'Missing or invalid release checksum.'
		}

		$actualChecksum = (Get-FileHash -LiteralPath $download -Algorithm SHA256).Hash
		if ($actualChecksum -ne $checksum) {
			throw 'Release checksum verification failed.'
		}

		# Keep an existing installation intact if replacement fails (for example, while tap is running).
		if ([IO.File]::Exists($destination)) {
			# Windows PowerShell 5.1 otherwise converts a null backup path to an empty string.
			[IO.File]::Replace($download, $destination, [NullString]::Value)
		} else {
			[IO.File]::Move($download, $destination)
		}

		$userPath = [string][Environment]::GetEnvironmentVariable('Path', 'User')
		$userDirectories = @($userPath -split ';' | ForEach-Object { $_.Trim().TrimEnd('\') })
		if ($userDirectories -notcontains $directory) {
			$userPath = ($userPath.TrimEnd(';') + ';' + $directory).TrimStart(';')
			[Environment]::SetEnvironmentVariable('Path', $userPath, 'User')
		}

		$sessionDirectories = @($env:Path -split ';' | ForEach-Object { $_.Trim().TrimEnd('\') })
		if ($sessionDirectories -notcontains $directory) {
			$env:Path = ($env:Path + ';' + $directory).TrimStart(';')
		}

		Write-Host "Installed tap $($release.tag_name) to $destination"
		Write-Host 'Run tap --help to get started. New terminals will also have tap on PATH.'
	} finally {
		[Net.ServicePointManager]::SecurityProtocol = $securityProtocol
		if (Test-Path -LiteralPath $staging) {
			Remove-Item -LiteralPath $staging -Recurse -Force
		}
	}
}
