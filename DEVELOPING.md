# Developing GoshuBNC

Most development happens on the `develop` branch, which is occasionally rebased + merged into `master` when it's not incredibly broken. When this happens, the `develop` branch is usually pruned until I feel like making 'unsafe' changes again.

I may also name the branch `develop+feature` if I'm developing multiple, or particularly unstable, features.

The intent is to keep `master` relatively stable.


## Updating `vendor/`

The `vendor/` directory holds our dependencies. When we import new repos, we need to update this folder to contain these new deps. This is something that I'll mostly be handling.

To update this folder:

1. Install https://github.com/golang/dep
2. `cd` to GoshuBNC folder
3. `dep ensure -update`
4. `cd vendor`
5. Commit the changes with the message `"Updated packages"`
6. `git push origin master`
7. `cd ..`
8. Commit the result with the message `"vendor: Updated submodules"`
9. Push the commit to the repo.

This will make sure things stay nice and up-to-date for users.


## Debugging Hangs

To debug a hang, the best thing to do is to get a stack trace. Go's nice, and you can do so by running this:

    $ kill -ABRT <procid>

This will kill GoshuBNC and print out a stack trace for you to take a look at.
