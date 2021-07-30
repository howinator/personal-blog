---
title: "We Need GPL for the Climate"
date: 2020-09-14T22:21:49-05:00
draft: true
categories: ['personal', 'climate change']
tags: ['climate change']
slug: "gpl-climate"
---

# Motivation

I recently did a quick calculation to determine how much CO₂e would be emitted if I shipped a change which increased the size of a code artifact which we serve to our customers.
Based off [this article from the ACEEE](https://www.aceee.org/files/proceedings/2012/data/papers/0193-000409.pdf), I found that a 1kb change to this artifact would result in ~6 metric tons of CO₂e being emitted yearly into the atmosphere.
This was surprising to me.
I work on software - the tech industry doesn't cause climate change.

If you've talked to me about climate change, you know I'm a huge proponent for a carbon tax and dividend program.
Between [popular resistance](https://en.wikipedia.org/wiki/Yellow_vests_movement), and the ineptitude of the U.S. government, we shouldn't have high hopes for the passage of [such a measure](https://www.congress.gov/bill/116th-congress/house-bill/763).
Fortunately (unfortunately, perhaps), recent history has shown that social progress often comes from industry, not government.
If industry takes up the cause of limiting climate change, how would they do that most effectively?

I propose a policy which uses the mechanics of [the GNU General Public License (GPL)](https://www.gnu.org/licenses/gpl-3.0.en.html) and applies it to carbon neutrality.

# GPL

Before I get into the bones of the policy, let's discuss the GPL.
The goal of the GPL is to promote software which is free (as in freedom) to use, modify and share.

The GPL accomplishes this with a clever concept called "Copyleft."
From [wikipedia](https://en.wikipedia.org/wiki/Copyleft):
> Copyleft is the practice of granting the right to freely distribute and modify intellectual property with the requirement that the same rights be preserved in derivative works created from that property.

In a reductionist sense, if a piece of software has a GPL license, it is open-source and any other software which uses it must also be open source and use the GPL license.
This makes GPL "viral" - once a critical mass of software is GPL-licensed, a knock-on effect is created where many more pieces of software are GPL-licensed.

# GPL for the Climate

Before I describe what I mean by "GPL for the Climate", take this as a disclaimer that I am not a lawyer and I do not claim that I've thought through the exact details or verbiage of the policy.
I simply claim the general idea has merit.

Without burying the lede any further, I propose industry leaders should adopt the following policy:
1. The company will be carbon neutral within 5 years.
1. The company will only sign or renew contracts with vendors which also adopt this policy.

There are a couple key points of this policy which require further explanation.
First, there is a grace period built into the policy.
There is no expectation that a company become carbon neutral tomorrow.

Second, this only effects the buy-side of a company.
Signatories are free to sell the most environmentally problematic company on earth.
This policy will get zero traction if it affects a company's bottom-line.

## Chicken or the Egg

There's absolutely an initialization problem with this policy.
In a market where all players are equal size, what incentive does an individual actor have to adopt such a policy when no one has?

Fortunately, the U.S. economy is oligopolistic - there are a few key players which constitute the majority of market share.
Furthermore, those key players have the ability to build services for themselves when no vendor is willing to adopt the policy.

To solve the initialization problem, this initiative would absolutely require one major leader in an industry to adopt the policy first.


## What's in it for a company?

So, why would a company adopt this policy?
I will split the motivation for adopting this policy into two types of companies, industry leaders and mid-tier companies.
Once I've described this relationship, the second-order effects of this policy should be apparent.

### Industry Leaders

There are two primary motivations for major companies to adopt this policy.

1. If a company cares about market value long-term, it's in the company's best interest to limit climate change.
   If we experience a major ecological collapse, the total addressable market of that company will decrease and so will its shareholder's value.
   Furthermore, there is a major effort by investors to divest from companies which are environmentally problematic.
2. Virtue signaling is important.
   Most major companies in the U.S. are facing a crisis of confidence and adopting this initiative could be a major arrow in their quiver to demonstrate societal responsibility.

### Mid-Tier Companies

Assuming industry leaders have adopted this policy, the motivation for mid-tier companies is purely competition-based.
If a mid-tier company would like to sell into enterprises, adopting this policy is a major competitive advantage.
As an example, if Microsoft has adopted GPL for the Climate, and Stripe has also adopted GPL for the climate but Square has not, Stripe has a major competitive advantage over Square.

### Start-ups

Once you see the relationship between industry leaders and mid-tier companies, it's easy to see how this recurses down the economy.
Start-ups are motivated to adopt this policy if they want to sell into mid-tier companies much like mid-tier companies are motivated to adopt the policy if they want to sell into enterprises.


# Prior art

Setting the expectation that suppliers be sustainable has quite a bit of prior art in industry.

One example is [REI's product sustainability standards.](https://www.rei.com/assets/stewardship/sustainability/rei-product-sustainability-standards/live.pdf)
The basis of this standard is that REI will give preferred treatment to vendors which commit to REI's sustainability guidelines.

In REI's sustainability report, [they account for the greenhouse gas emissions of their value chain](https://www.rei.com/stewardship/climate-change), but it's not clear to me if that includes the greenhouse gas emissions of their vendors' value chain.
GPL for the Climate solves the problem of recursing down the supply chain of a company.

# Potential details

## Enforcement



Grace period for vendors?
Auditors?

# Problems

What about AWS, i.e., entrenched companies which mid-tier companies can't easily leave

# Further Work

Legalities, marketing, etc.



