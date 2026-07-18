// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

// Package wizard collects the dotty init answers: the profile, the
// repository paths, and the what-goes-on-this-machine-class selections. A
// profile that already has answers seeds every question's default with them,
// so a re-run walks the same interview to extend the profile; flags silence
// their question, and without a terminal the seeded answers are taken as-is.
// The wizard only asks — rendering, linking, and key setup act on the
// scaffold.Answers it returns.
package wizard
