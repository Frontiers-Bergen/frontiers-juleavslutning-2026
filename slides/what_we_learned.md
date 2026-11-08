# What we learned this week

No software survives contact with its users.

- **6 people played tug of war:** the teams felt unfair, and it was hard to see your side
- **A load test with 160 phones:** 70% of the balloons got lost
- **A window on another workspace:** all the balloons took off at once when you came back

All three are fixed. We only found them by trying.

Notes:

- The user test: 6 people, very engaged. The teams were even among everyone who had ever opened the slide,
  including people who had left. Now new teams are drawn every round, among the phones watching, and your team
  stands in big letters above your end of the rope.
- The load test is a small Go program in the repo (`go tool task loadtest`) that runs locally: simulated phones
  smash during a tug of war round. Streams that fell behind lost balloons; now they take along what is already waiting.
- The browser does not draw a window on another workspace, but it still receives balloons. Now they fly by the clock.
- Test with people, measure, fix: that is what tonight is for too.
