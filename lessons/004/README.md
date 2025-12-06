# POSADEV DEMO

Service Discovery tool escrito con Go para el evento de PosaDev en GDL 2025 !

La idea, es demostrar cómo el lenguaje de programación Go te permite construir herramientas robustas y profesionales para resolver problemas de infraestructura.

Vamos a construir y deployar una herramienta de "Service Discovery" que observa los deployments en un custer de Kubernetes, y que escribe la información a un dashboard en un S3 bucket.

## High level architecture overview
<img width="992" height="490" alt="high-level-diagram-1" src="https://github.com/user-attachments/assets/10bab742-bf6e-4756-96f5-28e53bda4001" />


## Descripción General 

Es un sistema de **Service Discovery en tiempo real para Kubernetes** que simplifica la gestión de tu plataforma. Funciona con dos componentes clave:

* Un **backend en Go (go-sentinel)** que observa los *Deployments* del clúster y publica el estado de los servicios críticos directamente a S3.
* Un **frontend en JavaScript** que simplemente lee S3 y te presenta esa información en un dashboard visual y dinámico.


<img width="1024" height="642" alt="k8s-level-diagram" src="https://github.com/user-attachments/assets/49fe9fa8-64ea-4506-9400-bc503b3dd41d" />



PDF con los slides presentados en la charla:
[go-k8-charla.pdf](https://github.com/user-attachments/files/23994478/go-k8-charla.pdf)
