# ShakeSearch

Welcome to the Pulley Shakesearch Take-home Challenge! In this repository,
you'll find a simple web app that allows a user to search for a text string in
the complete works of Shakespeare.

You can see a live version of the app at
https://shakesearch-anudeep.herokuapp.com/. Try searching for "Hamlet" to display
a set of results.

In it's current state, however, the app is just a rough prototype. The search is
case sensitive, the results are difficult to read, and the search is limited to
exact matches.

## Your Mission

Improve the search backend. Think about the problem from the user's perspective
and prioritize your changes according to what you think is most useful.

## Submission

1. Fork this repository and send us a link to your fork after pushing your changes. 
2. Heroku hosting - The project includes a Heroku Procfile and, in its
current state, can be deployed easily on Heroku's free tier.
3. In your submission, share with us what changes you made and how you would prioritize changes if you had more time.

## Changes Made
1. Search function (attempts to) return a full line of dialogue, plus any relevant stage directions for plays, or the full sonnet the snippet appears in.
2. Search results are clearly divided into Sonnets and Plays on the results screen.
3. Backend search is parallelized - sonnets and plays are scoured in separate goroutines, and each "hit" spawns another goroutine to find the dialogeor sonnet boundaries, preventing any one hit waiting on a particularly adversarial boundary search, and brings up results without making the user stare at a blank screen for seconds at a time.

## Changes I would make given infinite time (ordered highest to lowest priority)
1. Half of Shakespeare's plays are in a different written format from the other half.  This means that, for those differently formatted plays, my search function pulls entire scenes instead of individual dialogue.  There are two ways to fix this - either write a go function to pore through the text file and standardize every play into the same format, or further index the search by each each work listed in the table of contents (as an extension of the existing sonnet/play indexing).  Ideally, I would do both, but it would make the backend search far easier to maintain if I were to focus on standardizing all the work, and enforcing that standardization on any future works that might be discovered and added.

2. The search currently cannot find an input phrase if it is split up across multiple lines.  It is also still case and punctuation sensitive.  Fixing these would have an approximately eaqual priority.

3. Finally, I would also add in a way to convert close matches of words - for example, Shakespeare often uses constructs like "cover'd" instead of "covered" in his works, presumably to help maintain his meter in the line.  So, I would want to add in a way for the search to match on truncated words and other tricks like that.


