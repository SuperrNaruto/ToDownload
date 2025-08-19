# GitHub Actions Workflows Recovery Guide

## Overview
The GitHub Actions workflows were temporarily removed from the initial commit due to OAuth permission restrictions. This guide explains how to restore them manually.

## Workflows to Restore

### 1. Update Contributors Workflow (`.github/workflows/update-contributors.yml`)
- **Purpose**: Automate contributor list updates in README
- **Trigger**: Manual workflow dispatch
- **Action**: Uses `akhilmhdh/contributors-readme-action@v2.3.10`

### 2. Documentation Deployment Workflow (`.github/workflows/docs.yml`)
- **Purpose**: Deploy Hugo documentation to GitHub Pages
- **Trigger**: Push to main branch with changes in `docs/` directory
- **Features**: Hugo site building and deployment to gh-pages branch

### 3. Release Build Workflow (`.github/workflows/build-release.yml`)
- **Purpose**: Build and release Go binaries for multiple platforms
- **Trigger**: Git tags starting with `v*`
- **Platforms**: Linux, macOS, Windows (amd64, arm64)
- **Features**: Automatic changelog generation and GitHub release creation

### 4. Docker Build Workflow (`.github/workflows/build-docker.yml`)
- **Purpose**: Build and publish Docker images to GitHub Container Registry
- **Trigger**: Git tags starting with `v*`
- **Platforms**: Linux/amd64, Linux/arm64
- **Registry**: GHCR (GitHub Container Registry)

## Restoration Steps

### Method 1: Direct File Creation
1. Create each workflow file manually in the GitHub web interface
2. Copy the content from the local `.github/workflows/` directory
3. Commit the files through the web interface

### Method 2: Using GitHub CLI (with proper permissions)
```bash
# Ensure you have the necessary permissions
gh auth status

# Create and push workflows
git add .github/workflows/
git commit -m "feat: restore GitHub Actions workflows"
git push origin main
```

### Method 3: Using Personal Access Token
1. Create a personal access token with `workflow` scope
2. Configure git to use the token:
```bash
git remote set-url origin https://YOUR_TOKEN@github.com/SuperrNaruto/ToDownload.git
```
3. Push the workflows:
```bash
git add .github/workflows/
git commit -m "feat: restore GitHub Actions workflows"
git push origin main
```

## Required Permissions

For the workflows to function properly, ensure the repository has:
- **Actions permissions**: Enabled
- **GitHub Pages permissions**: Enabled (for docs workflow)
- **Packages permissions**: Enabled (for Docker workflow)
- **Contents read & write**: For all workflows

## Security Considerations

The workflows are configured with minimal required permissions:
- `contents: write` - For repository operations
- `packages: write` - For container registry publishing
- `pull-requests: write` - For contributor updates

All workflows use official GitHub Actions and follow security best practices.

## Next Steps

1. Choose your preferred restoration method
2. Restore the workflows
3. Test the workflows by triggering them manually
4. Configure any necessary secrets (if required by the workflows)

## Files Available

The workflow files are available in the local `.github/workflows/` directory:
- `update-contributors.yml`
- `docs.yml`
- `build-release.yml`
- `build-docker.yml`

Each file contains the complete configuration and can be copied directly to GitHub.