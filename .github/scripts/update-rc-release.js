module.exports = async ({github, context, core}) => {
  const new_version = process.env.new_version;
  const bump_info = process.env.bump_info;
  
  try {
    const releases = await github.rest.repos.listReleases({
      owner: context.repo.owner,
      repo: context.repo.repo
    });
    
    const rcRelease = releases.data.find(release => release.tag_name.includes('-rc.'));
    
    const releaseBody = `This is the latest release candidate.

    Version: ${new_version}
    Commit: ${context.sha}
    BUMP_INFO: ${bump_info}`;
    
    if (rcRelease) {
      console.log(`Updating existing RC release: ${rcRelease.tag_name} to ${new_version}`);
      await github.rest.repos.updateRelease({
        owner: context.repo.owner,
        repo: context.repo.repo,
        release_id: rcRelease.id,
        tag_name: `v${new_version}`,
        name: `Release Candidate ${new_version}`,
        body: releaseBody,
        prerelease: true
      });
    } else {
      console.log(`Creating new RC release: ${new_version}`);
      await github.rest.repos.createRelease({
        owner: context.repo.owner,
        repo: context.repo.repo,
        tag_name: `v${new_version}`,
        name: `Release Candidate ${new_version}`,
        body: releaseBody,
        prerelease: true
      });
    }

    try {
      await github.rest.git.createRef({
        owner: context.repo.owner,
        repo: context.repo.repo,
        ref: `refs/tags/v${new_version}`,
        sha: context.sha
      });
    } catch (error) {
      if (error.status === 422) {
        await github.rest.git.updateRef({
          owner: context.repo.owner,
          repo: context.repo.repo,
          ref: `tags/v${new_version}`,
          sha: context.sha,
          force: true
        });
      } else {
        throw error;
      }
    }
  } catch (error) {
    console.error('Error updating or creating RC release:', error);
    core.setFailed(error.message);
  }
};
