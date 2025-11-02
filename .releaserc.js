module.exports = {
  branches: ['main'],
  plugins: [
    '@semantic-release/commit-analyzer',
    '@semantic-release/release-notes-generator',
    '@semantic-release/changelog',
    [
      '@semantic-release/exec',
      {
        prepareCmd: `
          yq e -i '.version = "${nextRelease.version}"' helm/Chart.yaml &&
          yq e -i '.appVersion = "${nextRelease.version}"' helm/Chart.yaml &&
          yq e -i '.image.tag = "${nextRelease.version}"' helm/values.yaml
        `,
      },
    ],
    [
      '@semantic-release/git',
      {
        assets: ['helm/Chart.yaml', 'helm/values.yaml', 'CHANGELOG.md'],
        message: 'chore(release): ${nextRelease.version} [skip ci]\n\n${nextRelease.notes}',
      },
    ],
    '@semantic-release/github',
  ],
};
