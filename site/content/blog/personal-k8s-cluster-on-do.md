---
title: "Personal Kubernetes Cluster on Digital Ocean With kubeadm"
date: 2019-01-23T22:49:40-06:00
draft: true
slug: k8s-on-do
categories: ['tech']
tags: ['tech', 'k8s', 'kubeadm', 'digital ocean']
---

# Introduction

I finally got around to bootstrapping a k8s cluster myself.
I really had two motivations for doing this:

1. I wanted a k8s cluster that I could easily deploy new personal projects to.
2. I wanted a k8s cluster to hack on so I could further my understanding of k8s.

I am now at at a point where I have a cluster which enables those motivations, so I decided it's time to write a blog post about it (served from the k8s-cluster itself :sunglasses:).

# Decisions, Decisions

Before I went into this, I had to make a couple design decisions to make.

## Infra Provider

My first decision was which infrastructure provider I'd build this on.
This decision was primarily driven by cost.

There were four main providers I looked at: GCP, AWS, DigitalOcean (DO), and OVH.
My assumptions for measuring pricing were that I'd mainly use machines with 2 CPUs and 4GB of RAM and the machines would be the latest generation if applicable.
I've tabulated the prices per month in the table below.

| AWS (t2.medium) | GCP (custom) | DO (standard droplet) | OVH (B2-7, 7GB RAM) |
|-----------------|--------------|-----------------------|---------------------|
| $29.95          | $43.39       | $20                   | $26.4               |



DigitalOcean is the clear winner in terms of pricing.

I won't lie - I was wishing that GCP would've been more competitive since I love working in GCP (topic for another blog post?), but GCP just didn't make financial sense.
So, I decided to go with DO, and so far, it really hasn't left me wanting.

## Kubernetes Provisioning Methodology/Tooling

Next, I had to decide on how I'd actually provision this thing.
The clear winner here is `kubeadm`.
I'll give a quick rundown of the tools in the space and then explain why `kubeadm` is the winner.

`kubespray` is an Ansible-based tool for deploying k8s.
With 5,000 stars, `kubespray` looks promising, but Ansible and I have a fraught relationship, so `kubespray` is out.

Next up is `kops`.
I've used `kops` at work in the past with great success on `AWS`, but they only officially support AWS, so `kops` was out.

Another tool I found is `bootkube`.
`bootkube` is a tool to help k8s admins deploy a self-hosted k8s-cluster.
Honestly, I think this approach is incredibly intellectually satisfying.
Besides being logically sound, this self-hosted approach makes [installs, scale-outs, and upgrades](https://coreos.com/blog/self-hosted-kubernetes.html).
The kubernetes team has a [great document about self-hosting](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/cluster-lifecycle/self-hosted-kubernetes.md).

As much as I like this approach, I'm not planning on having a high-availability, multi-master cluster because of cost.
Since I don't have a multi-master cluster, reboots of my master would kill my cluster and require manual intervention.
I decided against this approach since I wanted a reliable cluster that will be robust in the face of me doing terrible things to the cluster.

Finally, there's `kubeadm`.
`kubeadm` has recently become the officially accepted tool for deploying Kubernetes clusters using community-accepted best practices.
In fact, the tool is a [main line component in the Kubernetes repo](https://github.com/kubernetes/kubernetes/tree/master/cmd/kubeadm).
`kubeadm` allows you to


