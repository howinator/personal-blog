---
title: "Joy of Building"
date: 2026-02-11T11:05:02-08:00
draft: false
categories: ['personal', 'essay']
tags: ['personal', 'ai', 'leadership']
slug: "joy-of-building"
---

Do you remember the first time you fell in love with computers? That moment is one of my fondest memories. I was 13 and had just gotten back from watching Star Wars Episode III for the second time with my best friend, Andy. Andy and I were sitting at my parents' Compaq doing what we had taken a great fondness to doing recently — editing wiki pages on Wookiepedia to replace instances of “C3PO” with “C3P{{< redacted text="****" tip="I'll give you a hint, it's phallic." >}}O" (I'll let you fill in the blanks). Like the 7 times before, we saw the familiar page informing us we had been banned, except this time it was permanent. 

Andy was distraught, but I had been preparing for this day. I told him I had used a thing called Google to figure out a way around the ban. With some latent fear that I was about to ruin my Dad’s fancy new computer, I downloaded a thing called “PuTTY,” connected to something called a “router” using some strange number with too many decimals and typed `ifdown WAN && sleep 10 && ifup WAN` then hit enter. Those 10 seconds were excruciating — I was convinced, while we furiously refreshed the page, that I had destroyed the computer, taken down the entirety of the internet and the FBI was about to raid my house. But then, after some time, surely closer to an eternity than 10 seconds, the page loaded and wouldn’t you know it? We were back in action.

From that point forward, it was war. Our enemies were the moderators of Wookiepedia (I’m so sorry SparqMan) and they were facing off against two 13 year olds who would stop at nothing to plaster "C3P{{< redacted text="****" tip="Why did you hover again? Of course it was C3PenisO.<br>This should be obvious." >}}O" all over their website. 

I would say things escalated. They would write a script to ban any IP that made an edit containing that string, so I would write a script to edit 100 pages at once. They would return 400 for POST requests which contained that phrase, so I would spell the phrase out by adding a single letter to the beginning of every section title. It even escalated to the point where they ended up banning the entire CIDR block for AT&T in, at least, Houston, maybe Texas — I could never figure out the exact scope.

## && ifup WAN

Looking back, this was immature (I still contend hilarious though), but something about that online war left me absolutely hooked. The fact that I, a 13 year old kid, could, from my room, have an effect on people all over the world was bonkers to me — legitimately transcendent. With time, I came to learn that this concept is called “leverage.” Computers were leverage and oh boy did I like leverage.

With maturity, that desire for leveraged vandalism was shaped into something more positive. Though jobs come with a lot of bullshit, I try to not lose sight of the fact that having an outsize impact on the world is what motivates me. Moving into management, I fell in love with the idea of helping people grow in their ability to use the magic of computing to apply leverage to the world. Watching someone I care about growing from crippling impostor syndrome to owning a critical product area is more rewarding than e-vandalism ever was, but the point remains — all the margin you’ll give me and call me Bear Stearns, we’re hopping on the leverage train.

## sleep 10

Why am I writing this? 
I'm writing this because experiencing the LLM transition has brought up deep questions around where I can apply leverage.
Is it code? Claude code takes care of that now. Is it people & teams? Where can I as an engineering leader actually apply leverage? The answer isn't obvious.

Recently, specifically while I typed this essay, I've convinced myself nothing has fundamentally changed here.
I still apply leverage by building. The things I build might change, but as an engineer, I create leverage by building.
And in fact, the things I build are even more leveraged. 

I recently wanted to build a drone for tracking the neighborhood bear as a way to lightly shame my neighbors for not securing their trash cans.
Without AI, I'd have to learn the minutiae of flight controllers, AI vision and control theory. All that just to stop a bear from gorging on 5 Costco rotisserie chicken carcasses? Yeah good one.

With AI? That suddenly seems worth it — no one wants our bear getting an unfortunate nickname[^1]. And what does it take from me? Just some naive curiosity and willingness to build.

If you're an engineer worried about building, this can only be exciting. Imagine what a team of extremely leveraged builders could create. An entire society operating in this way? I can only imagine where we end up — getting C3P{{< redacted text="****" tip="I get it. I like hovering over stuff too." >}}O on the real Wikipedia? A boy can dream.

## Furiously Refreshing

I've found excitement often comes with a healthy pairing of fear. And there's plenty of fear about what lies between here and there. When I was 13 and experiencing the joy of computing, it absolutely seemed to me like I was going to break the computer, the whole internet, in fact, and do my middle school graduation from a jail cell. 

Sure I broke a few eggs and had to explain to my Mom why the internet wasn’t working. But did the worst case happen? Of course not. It’s absurd in retrospect. And I’ve been rewarded with an incredible career of making the world a better place by building teams and computer programs. 

Broadly in tech right now, I yearn for that same adolescent sense of curiosity about how we do work, the processes we use to collaborate and the products that we build. Instead, it’s been a steady drip of one part cynicism and two parts tribalism. 

I understand the stakes are higher for someone whose livelihood is on the line than a 13 year old boy trolling a Star Wars wiki. Is it really so different though? We’re all collectively children when it comes to this technology. And if any of these words have resonated with you so far, I guarantee it’s more exciting to get curious and do whatever you can to leverage this new form of computing than be too scared to download PuTTY and live a life without the joy of this career. 

Did I mention I didn't have an allowance? 
If my Mom is reading this, I haven’t forgotten. I’ll never forget. 
But also, would I have been so rebellious if I had one? Maybe not. I wouldn’t want to risk losing my ticket to Halo 2. That’s why being at a startup feels so unfairly enviable right now. I can rip up the playbook, try new ways of building software, be wrong and then learn the best way to build a technology company in an AI world without putting the non-existent allowance at risk. Being in an environment where you have permission to truly play with and break the new thing is no different than my parents letting me go wild with the new computer and high speed DSL internet. I was grateful then and I’m grateful now… okay, Mom, I forgive you. 

This time feels different. 
I have no idea what happens next week, next month, next year. 
What I do know is that along the way, we'll participate in the joy of building if we let it. 
And that's exactly what that 13 year old boy signed up for.

So please, go right ahead and download PuTTY, reset your IP address, run those parallel agents, merge LLM-generated PRs, revert that one LLM-generated PR, be scared, get excited, screw up, learn, but please, above all else, build the future with me — it's why we fell in love with computing.

[^1]: https://en.wikipedia.org/wiki/Hank_the_Tank
