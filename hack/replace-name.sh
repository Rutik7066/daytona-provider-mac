#!/bin/bash

# Check for macOS using system information
if [[ $(uname -s) == "Darwin" ]]; then
    SED_CMD="gsed"
    if ! command -v gsed &>/dev/null; then
        echo "gsed is not installed. It is required for this script to function properly."
        echo "Would you like to install gsed using Homebrew? (y/N)"
        read -r response

        if [[ $response =~ ^([Yy])$ ]]; then
            # Install gsed with Homebrew (if available)
            if command -v brew &>/dev/null; then
                brew install gsed
            else
                echo "Homebrew is not installed. Please install gsed manually or another package manager."
            fi
            exit 1
        else
            echo "Exiting script as gsed is required."
            exit 1
        fi
    fi
    echo "gsed is installed. Proceeding..."
else
    SED_CMD="sed"
fi

# Print and get input for repository name
echo -n "URL of your repository (without https://): "
read -r repositoryUrl

# Print and get input for provider name
echo -n "Name of the provider (e.g. Docker Provider): "
read -r providerName

# Replace ocurrences of github.com/rutik7066/daytona-provider-mac with the repository name
find . -type d \( -name "hack" -o -name ".git" \) -prune -o -type f -exec $SED_CMD -i "s|github.com/rutik7066/daytona-provider-mac|$repositoryUrl|g" {} +
echo "Replaced github.com/rutik7066/daytona-provider-mac with $repositoryUrl"

# Replace occurrences of "mac-provider" with formatted provider name
find . -type d \( -name "hack" -o -name ".git" \) -prune -o -type f -exec $SED_CMD -i "s/mac-provider/$(echo "$providerName" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')/g" {} +
echo "Replaced mac-provider with $(echo "$providerName" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')"

# Replace ocurrences of "MacProvider" with the provider name
find . -type d \( -name "hack" -o -name ".git" \) -prune -o -type f -exec $SED_CMD -i "s/MacProvider/$(echo "$providerName" | tr -d ' ')/g" {} +
echo "Replaced MacProvider with $(echo "$providerName" | tr -d ' ')"

# Replace occurrences of "Mac Provider" with the provider name
find . -type d \( -name "hack" -o -name ".git" \) -prune -o -type f -exec $SED_CMD -i "s/Mac Provider/$providerName/g" {} +
echo "Replaced Mac Provider with $providerName"
