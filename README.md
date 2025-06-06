
# EC2 MySQL And AWS EC2 Server Administrator - SSH Automation Toolkit

## Overview
A **Go-based CLI tool** that automates MySQL administration and Linux server management on AWS EC2 instances through secure SSH commands. Designed for DevOps and cloud engineers who need to streamline database operations without manual SSH sessions.

## Key Features

   **SSH Automation**  
- Secure connection handling with PEM keys  
- Concurrent command execution with output streaming  

    **MySQL Management**  
- User creation (full/backup privileges)  
- Bind address configuration  
- Database backup generation  

    **Server Operations**  
- File transfers via SCP  
- Sudo command execution  

## Technical Stack
- **Language**: Go 1.20+  
- **Libraries**: `golang.org/x/crypto/ssh`  
- **Infrastructure**: AWS EC2, MySQL 5.7+/8.0  

