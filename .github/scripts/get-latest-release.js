module.exports = async ({github, context}) => {
  try {
    const releases = await github.rest.repos.listReleases({
      owner: context.repo.owner,
      repo: context.repo.repo
    });
    const latestRelease = releases.data[0];
    if (!latestRelease) {
      return JSON.stringify({ version: '0.0.0', bumpInfo: { major: false, minor: false, patch: false } });
    }
    const version = latestRelease.tag_name.replace('v', '');
    const bumpInfoMatch = latestRelease.body.match(/BUMP_INFO: ({[^}]+})/);
    const bumpInfo = bumpInfoMatch ? JSON.parse(bumpInfoMatch[1]) : { major: false, minor: false, patch: false };
    return JSON.stringify({ version, bumpInfo });
  } catch (e) {
    console.error(e);
    return JSON.stringify({ version: '0.0.0', bumpInfo: { major: false, minor: false, patch: false } });
  }
};
