---
title: 'BioHackrXiv  template'
tags:
  - replace with your own keywords
  - at least 3 is recommended
authors:
  - name: Johan Viklund
    orcid: 0000-0003-1984-8522
    affiliation: 3
  - name: Stefan Negru
    orcid: 0000-0002-6544-5022
    affiliation: 1
  - name: Dimitris Bampalikis
    affiliation: 3
    orcid: 0000-0002-2078-3079
  - name: Liisa Lado-Villar
    orcid:
    affiliation: 1
  - name: Alexandros Dimopoulos
    orcid: 0000-0002-4602-2040
    affiliation: 2
  - name: Konstantinos Koumpouras
    orcid: 0000-0001-5598-4703
    affiliation: 3
  - name: Alexandros Aperis
    orcid: 0000-0002-1699-2476
    affiliation: 3
  - name:  Panagiotis Chatzopoulos
    orcid: 0009-0004-7445-2453
    affiliation: 3
  - name: Marko Malenic
    orcid: 0009-0007-3824-8449
    affiliation: 4

affiliations:
 - name: CSC – IT CENTER FOR SCIENCE, Espoo, Finland
    index: 1
  - name: Biomedical Sciences Research Center Alexander Fleming, Vari, Greece
    index: 2
  - name: National Bioinformatics Infrastructure Sweden (NBIS), Uppsala University, SciLifeLab, ICM - Department of Cell and Molecular Biology, Uppsala, Sweden.
    index: 3
  - name: University of Melbourne, Melbourne, AU 
    index: 4
date: 01 January 2020
bibliography: paper.bib
authors_short: Last et al. (2021) BioHackrXiv  template
group: BioHackrXiv
event: BioHackathon Europe 2021
---

# Introduction or Background

The European Genome-phenome Archive (EGA) [@EGA] is a service for archiving and sharing personally identifiable genetic and phenotypic data, while the The Genomic Data Infrastructure (GDI) [@GDI] project is enabling access to genomic and related phenotypic and clinical data across Europe. Both projects are focused on creating federated and secure infrastructure for researchers to archive and share data with the research community, to support further research.


The project was focused on the data access part of the infrastructure. The files are encrypted in the archives, using the crypt4gh standard [@crypt4gh]. Currently, there exist data access processes, where the files are either decrypted on the server side and then transferred to the user or re-encrypted server-side and provided to the user in an outbox.


Htsget [@htsget] as a data access protocol also allows access to parts of files. Before the Biohackathon event, there did not exist any production-level client tools that supports access to encrypted data. The main goal of the project was to create a client tool that can access encrypted data over the htsget protocol, able to work with the GA4GH Passport and Visa standard, which enhances the security of the data access interfaces.

## Subsection level 2

In order to enable for random data access on encrypted files, we worked on the htsget-rs [@htsget-rs], a Rust htsget server to support the aforementioned standards and the sda-download, an implementation handling the data-out API of the archives, developed by the Nordic collaboration under the umbrella of the Nordic e-Infrastructure Collaboration(NeIC) [@NEIC].


### Subsection level 3

Please keep sections to a maximum of three levels.

## Tables, figures and so on

Please remember to introduce tables (see Table 1) before they appear on the document. We recommend to center tables, formulas and figure but not the corresponding captions. Feel free to modify the table style as it better suits to your data.

Table 1
| Header 1 | Header 2 |
| -------- | -------- |
| item 1 | item 2 |
| item 3 | item 4 |

Remember to introduce figures (see Figure 1) before they appear on the document. 

![BioHackrXiv logo](./biohackrxiv.png)
 
Figure 1. A figure corresponding to the logo of our BioHackrXiv preprint.

# Other main section on your manuscript level 1

Feel free to use numbered lists or bullet points as you need.
* Item 1
* Item 2

# Discussion and/or Conclusion

We recommend to include some discussion or conclusion about your work. Feel free to modify the section title as it fits better to your manuscript.

# Future work

And maybe you want to add a sentence or two on how you plan to continue. Please keep reading to learn about citations and references.

For citations of references, we prefer the use of parenthesis, last name and year. If you use a citation manager, Elsevier – Harvard or American Psychological Association (APA) will work. If you are referencing web pages, software or so, please do so in the same way. Whenever possible, add authors and year. We have included a couple of citations along this document for you to get the idea. Please remember to always add DOI whenever available, if not possible, please provide alternative URLs. You will end up with an alphabetical order list by authors’ last name.

# Jupyter notebooks, GitHub repositories and data repositories

* Please add a list here
* Make sure you let us know which of these correspond to Jupyter notebooks. Although not supported yet, we plan to add features for them
* And remember, software and data need a license for them to be used by others, no license means no clear rules so nobody could legally use a non-licensed research object, whatever that object is

# Acknowledgements
Please always remember to acknowledge the BioHackathon, CodeFest, VoCamp, Sprint or similar where this work was (partially) developed.

# References

Leave thise section blank, create a paper.bib with all your references.
