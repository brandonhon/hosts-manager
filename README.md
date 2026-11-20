# hosts-manager website

This branch holds only the project website, served by GitHub Pages. The code,
docs and releases live on [`main`](https://github.com/brandonhon/hosts-manager).

- `index.html` is the whole site: one self-contained page with no build step.
  The editor and terminal demos run in the browser on sample data.
- The logo files are copies from `docs/images/logo/` on `main`; update them
  there first.
- `.nojekyll` tells GitHub Pages to serve the files as they are.

The install section asks the GitHub API for the latest release when the page
loads, so it does not need editing after a release. If that request fails it
shows the version written into `index.html`.

To preview, open `index.html` in a browser. Changes reach this branch through
pull requests.
