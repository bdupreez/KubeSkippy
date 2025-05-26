#!/bin/bash

# Replace 'yourusername' with your actual GitHub username
GITHUB_USERNAME="bdupreez"

echo "Adding GitHub remote..."
git remote add origin git@github.com:${GITHUB_USERNAME}/KubeSkippy.git

echo "Pushing to GitHub..."
git branch -M main
git push -u origin main

echo "Done! Your repository is now on GitHub."
echo "Don't forget to:"
echo "1. Update all 'yourusername' references in the code"
echo "2. Add DOCKER_USERNAME and DOCKER_PASSWORD secrets in GitHub Settings > Secrets"
