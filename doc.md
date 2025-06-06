# SSH MySQL Automation Toolkit - Core Functions

## SSH Connection Management

### `serverSetup(pemFilePath, passwordKey)`
- **Purpose**: Initialize SSH connection using PEM key  
- **Parameters**:  
  - `pemFilePath`: Path to .pem file  
  - `passwordKey`: Optional passphrase for encrypted keys  
- **Returns**: `ssh.Signer` or error  
- **Behavior**:  
  - Reads and parses private key  
  - Supports passphrase-protected keys  

### `createClient()`
- **Purpose**: Configure SSH client parameters  
- **Returns**: `ssh.ClientConfig` or error  
- **Features**:  
  - Sets 10s timeout  
  - Uses public key authentication  

### `createConnection()`
- **Purpose**: Establish SSH connection to EC2  
- **Returns**: Active `ssh.Client` connection or error  
- **Network**:  
  - TCP connection on port 22  
  - 10s dial timeout  

## Remote Command Execution

### `remoteCommand(cmd)`
- **Purpose**: Execute non-sudo commands remotely  
- **Features**:  
  - Concurrent stdout/stderr handling  
  - Real-time output streaming  
  - Automatic connection cleanup  

### `sudoCmd(command, sudoPassword)`
- **Purpose**: Execute privileged commands  
- **Security**:  
  - Password passed via secure stdin  
  - Output captured separately  

## MySQL Administration

### Database Configuration
| Function | Purpose | Key Feature |
|----------|---------|-------------|
| `changeBindAddrToGeneral()` | Change MySQL bind address | Updates `mysqld.cnf` → 0.0.0.0 |
| `findCurrentBindAddr()` | Detect current binding | Uses `SHOW VARIABLES` |
| `changeBindAddr()` | Modify bind address | Dynamic sed replacement |

### User Management
| Function | Privileges | Security |
|----------|------------|----------|
| `createUsersMysql()` | FULL DB privileges | Secure password handling |
| `createBackupUser()` | SELECT + LOCK TABLES | Least-privilege principle |

## Data Operations

### `moveFilesToTheServer(filePath)`
- **Protocol**: SCP transfer  
- **Validation**: File existence checks  

### `moveMysqlDBToTheServer()`
- **Workflow**:  
  1. Local `mysqldump` execution  
  2. Secure SCP transfer to EC2  

### `createBackup()`
- **Output**: Timestamped SQL file  
- **Naming**: `YYYY-MM-DD_HH-MM-SS` format  

## Technical Specifications
- **Concurrency**: Goroutines for async output handling  
- **Error Handling**: Wrapped errors with context  
- **Security**:  
  - PEM key authentication  
  - Minimal sudo exposure  
