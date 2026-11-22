# Not-found checks on wrapped errors need `errors.Is`, not `os.IsNotExist`

`os.IsNotExist` deliberately does not traverse `%w` wrap chains, so a
not-found tolerance on an error that has crossed a package seam is
silently dead code; use `errors.Is(err, os.ErrNotExist)`.

Go's `os.IsNotExist(err)` inspects only the error it is handed; it
deliberately does not traverse `%w` wrapping chains — the stdlib
source says so outright ("this function is not errors.Is"). The
error-wrapping convention compounds the exposure: cross-package errors
are wrapped with `fmt.Errorf("context: %w", err)` (see
[Error Wrapping Context](../../errors/rules/error_wrapping.md)), so any
error that has crossed a package seam can carry a wrap. A not-found
tolerance written as `!os.IsNotExist(err)` against such an error never
matches — the tolerance is silently dead code: no error, no log; the
guard branch simply never runs. The fix is
`errors.Is(err, os.ErrNotExist)` (`os.ErrNotExist` and `fs.ErrNotExist`
are the same sentinel; both spellings are correct).

## The costed incident

In one observed case the dead `os.IsNotExist` check sat for months in
a daemon's crash-recovery test leg — a kill-9 scenario whose baseline
tolerance accepted a record that was either valid or absent. It
surfaced only when a commit-ordering change made an absent record
actually reachable at kill time. That is the signature of the class:
the defect is invisible not because the code is unexercised in general
but because the not-found path specifically is unreachable until some
unrelated behavior change opens it.

## How to apply

Whenever a tolerance or branch guards a not-found condition on an
error crossing a package seam — wrapping is convention there — grep
for `os.IsNotExist`: if the error variable can carry a wrap, switch to
`errors.Is(err, os.ErrNotExist)`. Bare `os.IsNotExist` remains
correct only on unwrapped `os.Stat`-style returns consumed at their
own call site.

Regression-testing the class requires seam reachability: a
dead-tolerance bug reds only when the fixture drives the wrapped
not-found path itself. That is the same reachability discipline the
engineering module's bug-diagnosis procedure demands of a correct
regression seam (its fix-behind-a-regression-test phase: a seam too
shallow to replicate the triggering chain produces false confidence,
and reachability is half the seam).
