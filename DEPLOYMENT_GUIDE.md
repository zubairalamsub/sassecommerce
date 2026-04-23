# Deployment & DevOps Guide

## Overview

This guide covers the complete DevOps strategy for deploying and managing the e-commerce platform, including CI/CD pipelines, infrastructure as code, containerization, and operational procedures.

---

## Infrastructure as Code (IaC)

### Terraform Configuration

#### AWS Infrastructure Setup

```hcl
# terraform/main.tf

terraform {
  required_version = ">= 1.0"

  backend "s3" {
    bucket         = "ecommerce-terraform-state"
    key            = "production/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-lock"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# VPC Configuration
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  name = "ecommerce-vpc"
  cidr = "10.0.0.0/16"

  azs             = ["us-east-1a", "us-east-1b", "us-east-1c"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]

  enable_nat_gateway = true
  enable_vpn_gateway = false

  tags = {
    Environment = var.environment
    Project     = "ecommerce"
  }
}

# EKS Cluster
module "eks" {
  source = "terraform-aws-modules/eks/aws"

  cluster_name    = "ecommerce-${var.environment}"
  cluster_version = "1.28"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  eks_managed_node_groups = {
    general = {
      desired_size = 3
      min_size     = 2
      max_size     = 10

      instance_types = ["t3.xlarge"]
      capacity_type  = "ON_DEMAND"

      labels = {
        role = "general"
      }
    }

    compute = {
      desired_size = 2
      min_size     = 1
      max_size     = 5

      instance_types = ["c5.2xlarge"]
      capacity_type  = "SPOT"

      labels = {
        role = "compute-intensive"
      }

      taints = [{
        key    = "compute"
        value  = "true"
        effect = "NoSchedule"
      }]
    }
  }

  tags = {
    Environment = var.environment
  }
}

# RDS PostgreSQL
resource "aws_db_instance" "postgres" {
  identifier = "ecommerce-postgres-${var.environment}"

  engine         = "postgres"
  engine_version = "15.3"
  instance_class = "db.r6g.xlarge"

  allocated_storage     = 100
  max_allocated_storage = 1000
  storage_encrypted     = true

  db_name  = "ecommerce"
  username = var.db_username
  password = var.db_password

  multi_az               = true
  backup_retention_period = 7
  backup_window          = "03:00-04:00"
  maintenance_window     = "mon:04:00-mon:05:00"

  vpc_security_group_ids = [aws_security_group.rds.id]
  db_subnet_group_name   = aws_db_subnet_group.main.name

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]

  tags = {
    Environment = var.environment
  }
}

# ElastiCache Redis
resource "aws_elasticache_replication_group" "redis" {
  replication_group_id       = "ecommerce-redis-${var.environment}"
  replication_group_description = "Redis cluster for caching"

  engine         = "redis"
  engine_version = "7.0"
  node_type      = "cache.r6g.large"

  num_cache_clusters         = 3
  automatic_failover_enabled = true
  multi_az_enabled          = true

  subnet_group_name  = aws_elasticache_subnet_group.main.name
  security_group_ids = [aws_security_group.redis.id]

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true

  snapshot_retention_limit = 5
  snapshot_window         = "03:00-05:00"

  tags = {
    Environment = var.environment
  }
}

# MSK Kafka Cluster
resource "aws_msk_cluster" "kafka" {
  cluster_name           = "ecommerce-kafka-${var.environment}"
  kafka_version          = "3.5.1"
  number_of_broker_nodes = 3

  broker_node_group_info {
    instance_type   = "kafka.m5.large"
    client_subnets  = module.vpc.private_subnets
    security_groups = [aws_security_group.kafka.id]

    storage_info {
      ebs_storage_info {
        volume_size = 100
      }
    }
  }

  encryption_info {
    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
    encryption_at_rest_kms_key_arn = aws_kms_key.kafka.arn
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.kafka.name
      }
    }
  }

  tags = {
    Environment = var.environment
  }
}

# S3 Bucket for Assets
resource "aws_s3_bucket" "assets" {
  bucket = "ecommerce-assets-${var.environment}"

  tags = {
    Environment = var.environment
  }
}

resource "aws_s3_bucket_versioning" "assets" {
  bucket = aws_s3_bucket.assets.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "assets" {
  bucket = aws_s3_bucket.assets.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# CloudFront Distribution
resource "aws_cloudfront_distribution" "assets" {
  enabled             = true
  is_ipv6_enabled     = true
  price_class         = "PriceClass_All"
  default_root_object = "index.html"

  origin {
    domain_name = aws_s3_bucket.assets.bucket_regional_domain_name
    origin_id   = "S3-${aws_s3_bucket.assets.id}"

    s3_origin_config {
      origin_access_identity = aws_cloudfront_origin_access_identity.assets.cloudfront_access_identity_path
    }
  }

  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD", "OPTIONS"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3-${aws_s3_bucket.assets.id}"

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "redirect-to-https"
    min_ttl                = 0
    default_ttl            = 3600
    max_ttl                = 86400
    compress               = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  tags = {
    Environment = var.environment
  }
}
```

---

## Kubernetes Configuration

### Namespace Setup

```yaml
# k8s/namespaces/production.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    environment: production
---
apiVersion: v1
kind: Namespace
metadata:
  name: staging
  labels:
    environment: staging
```

### Service Deployment Example

```yaml
# k8s/services/user-service/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
  namespace: production
  labels:
    app: user-service
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: user-service
  template:
    metadata:
      labels:
        app: user-service
        version: v1
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "3000"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: user-service

      containers:
      - name: user-service
        image: 123456789.dkr.ecr.us-east-1.amazonaws.com/user-service:latest
        imagePullPolicy: Always

        ports:
        - name: http
          containerPort: 3000
          protocol: TCP

        env:
        - name: NODE_ENV
          value: "production"
        - name: PORT
          value: "3000"
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: user-service-secrets
              key: db-host
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: user-service-secrets
              key: db-password
        - name: REDIS_URL
          valueFrom:
            configMapKeyRef:
              name: user-service-config
              key: redis-url

        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"

        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3

        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3

        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 15"]

      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - user-service
              topologyKey: kubernetes.io/hostname
---
apiVersion: v1
kind: Service
metadata:
  name: user-service
  namespace: production
  labels:
    app: user-service
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: http
    protocol: TCP
    name: http
  selector:
    app: user-service
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: user-service
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: user-service
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 30
      - type: Pods
        value: 4
        periodSeconds: 30
      selectPolicy: Max
```

### ConfigMap

```yaml
# k8s/services/user-service/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: user-service-config
  namespace: production
data:
  redis-url: "redis://ecommerce-redis.cache.amazonaws.com:6379"
  log-level: "info"
  rate-limit-max: "100"
  rate-limit-window: "60000"
```

### Secrets

```yaml
# k8s/services/user-service/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: user-service-secrets
  namespace: production
type: Opaque
stringData:
  db-host: "ecommerce-postgres.rds.amazonaws.com"
  db-username: "user_service"
  db-password: "${DB_PASSWORD}" # Injected via CI/CD
  jwt-secret: "${JWT_SECRET}"
```

---

## Istio Service Mesh Configuration

### Virtual Service

```yaml
# k8s/istio/user-service-virtualservice.yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: user-service
  namespace: production
spec:
  hosts:
  - user-service
  http:
  - match:
    - headers:
        x-api-version:
          exact: v2
    route:
    - destination:
        host: user-service
        subset: v2
      weight: 100
  - route:
    - destination:
        host: user-service
        subset: v1
      weight: 90
    - destination:
        host: user-service
        subset: v2
      weight: 10
  - timeout: 10s
    retries:
      attempts: 3
      perTryTimeout: 2s
      retryOn: 5xx,reset,connect-failure,refused-stream
```

### Destination Rule

```yaml
# k8s/istio/user-service-destinationrule.yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: user-service
  namespace: production
spec:
  host: user-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 50
        http2MaxRequests: 100
        maxRequestsPerConnection: 2
    loadBalancer:
      simple: LEAST_REQUEST
    outlierDetection:
      consecutiveErrors: 5
      interval: 30s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
```

---

## CI/CD Pipeline

### GitHub Actions - Complete Workflow

```yaml
# .github/workflows/deploy.yml
name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  AWS_REGION: us-east-1
  ECR_REGISTRY: 123456789.dkr.ecr.us-east-1.amazonaws.com
  EKS_CLUSTER: ecommerce-production

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379

    steps:
    - uses: actions/checkout@v3

    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'
        cache: 'npm'

    - name: Install dependencies
      run: npm ci

    - name: Run linter
      run: npm run lint

    - name: Run unit tests
      run: npm run test:unit

    - name: Run integration tests
      run: npm run test:integration
      env:
        DATABASE_URL: postgresql://postgres:postgres@localhost:5432/test
        REDIS_URL: redis://localhost:6379

    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        files: ./coverage/coverage-final.json

    - name: SonarCloud Scan
      uses: SonarSource/sonarcloud-github-action@master
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}

  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.event_name == 'push'

    strategy:
      matrix:
        service: [user-service, product-service, order-service, payment-service]

    steps:
    - uses: actions/checkout@v3

    - name: Configure AWS credentials
      uses: aws-actions/configure-aws-credentials@v2
      with:
        aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
        aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        aws-region: ${{ env.AWS_REGION }}

    - name: Login to Amazon ECR
      id: login-ecr
      uses: aws-actions/amazon-ecr-login@v1

    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v2

    - name: Build and push Docker image
      uses: docker/build-push-action@v4
      with:
        context: ./services/${{ matrix.service }}
        push: true
        tags: |
          ${{ env.ECR_REGISTRY }}/${{ matrix.service }}:${{ github.sha }}
          ${{ env.ECR_REGISTRY }}/${{ matrix.service }}:latest
        cache-from: type=gha
        cache-to: type=gha,mode=max

    - name: Scan image for vulnerabilities
      uses: aquasecurity/trivy-action@master
      with:
        image-ref: ${{ env.ECR_REGISTRY }}/${{ matrix.service }}:${{ github.sha }}
        format: 'sarif'
        output: 'trivy-results.sarif'

    - name: Upload Trivy results to GitHub Security
      uses: github/codeql-action/upload-sarif@v2
      with:
        sarif_file: 'trivy-results.sarif'

  deploy-staging:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/develop'
    environment: staging

    steps:
    - uses: actions/checkout@v3

    - name: Configure AWS credentials
      uses: aws-actions/configure-aws-credentials@v2
      with:
        aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
        aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        aws-region: ${{ env.AWS_REGION }}

    - name: Update kubeconfig
      run: |
        aws eks update-kubeconfig --name ecommerce-staging --region ${{ env.AWS_REGION }}

    - name: Deploy to Kubernetes
      run: |
        kubectl set image deployment/user-service \
          user-service=${{ env.ECR_REGISTRY }}/user-service:${{ github.sha }} \
          -n staging
        kubectl rollout status deployment/user-service -n staging

    - name: Run smoke tests
      run: npm run test:smoke
      env:
        API_URL: https://staging-api.example.com

  deploy-production:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    environment: production

    steps:
    - uses: actions/checkout@v3

    - name: Configure AWS credentials
      uses: aws-actions/configure-aws-credentials@v2
      with:
        aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
        aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        aws-region: ${{ env.AWS_REGION }}

    - name: Update kubeconfig
      run: |
        aws eks update-kubeconfig --name ${{ env.EKS_CLUSTER }} --region ${{ env.AWS_REGION }}

    - name: Deploy with ArgoCD
      run: |
        argocd app set user-service --helm-set-string image.tag=${{ github.sha }}
        argocd app sync user-service --prune
        argocd app wait user-service --health

    - name: Run E2E tests
      run: npm run test:e2e
      env:
        API_URL: https://api.example.com

    - name: Notify Slack
      uses: 8398a7/action-slack@v3
      with:
        status: ${{ job.status }}
        text: 'Deployment to production completed!'
        webhook_url: ${{ secrets.SLACK_WEBHOOK }}
      if: always()
```

---

## ArgoCD Configuration

### Application Definition

```yaml
# argocd/user-service.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: user-service
  namespace: argocd
spec:
  project: ecommerce

  source:
    repoURL: https://github.com/example/ecommerce-platform
    targetRevision: HEAD
    path: k8s/services/user-service
    helm:
      valueFiles:
      - values-production.yaml

  destination:
    server: https://kubernetes.default.svc
    namespace: production

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
      allowEmpty: false
    syncOptions:
    - Validate=true
    - CreateNamespace=false
    - PrunePropagationPolicy=foreground
    - PruneLast=true

    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m

  revisionHistoryLimit: 10
```

---

## Database Migration Strategy

### Flyway Migration Script

```sql
-- migrations/V1__initial_schema.sql
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

### Migration Job

```yaml
# k8s/jobs/db-migration.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migration-v1
  namespace: production
spec:
  template:
    spec:
      restartPolicy: OnFailure
      containers:
      - name: flyway
        image: flyway/flyway:9
        args:
        - migrate
        env:
        - name: FLYWAY_URL
          value: "jdbc:postgresql://ecommerce-postgres.rds.amazonaws.com:5432/ecommerce"
        - name: FLYWAY_USER
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: username
        - name: FLYWAY_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: password
        volumeMounts:
        - name: migrations
          mountPath: /flyway/sql
      volumes:
      - name: migrations
        configMap:
          name: db-migrations
```

---

## Monitoring Setup

### Prometheus Configuration

```yaml
# k8s/monitoring/prometheus-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s

    scrape_configs:
    - job_name: 'kubernetes-pods'
      kubernetes_sd_configs:
      - role: pod
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
```

### Grafana Dashboards

```json
{
  "dashboard": {
    "title": "User Service Metrics",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total{service=\"user-service\"}[5m])"
          }
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total{service=\"user-service\",status=~\"5..\"}[5m])"
          }
        ]
      },
      {
        "title": "Response Time (p95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{service=\"user-service\"}[5m]))"
          }
        ]
      }
    ]
  }
}
```

---

## Disaster Recovery Procedures

### Backup Strategy

```bash
#!/bin/bash
# scripts/backup-postgres.sh

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="backup_${TIMESTAMP}.sql"

# Backup database
pg_dump -h $DB_HOST -U $DB_USER -d ecommerce > $BACKUP_FILE

# Compress
gzip $BACKUP_FILE

# Upload to S3
aws s3 cp ${BACKUP_FILE}.gz s3://ecommerce-backups/postgres/${BACKUP_FILE}.gz

# Cleanup old backups (keep 30 days)
aws s3 ls s3://ecommerce-backups/postgres/ | \
  while read -r line; do
    createDate=$(echo $line | awk {'print $1" "$2'})
    createDate=$(date -d "$createDate" +%s)
    olderThan=$(date --date "30 days ago" +%s)
    if [[ $createDate -lt $olderThan ]]; then
      fileName=$(echo $line | awk {'print $4'})
      aws s3 rm s3://ecommerce-backups/postgres/$fileName
    fi
  done
```

### Restore Procedure

```bash
#!/bin/bash
# scripts/restore-postgres.sh

BACKUP_FILE=$1

# Download from S3
aws s3 cp s3://ecommerce-backups/postgres/$BACKUP_FILE .

# Decompress
gunzip $BACKUP_FILE

# Restore
psql -h $DB_HOST -U $DB_USER -d ecommerce < ${BACKUP_FILE%.gz}
```

---

## Security Hardening

### Network Policies

```yaml
# k8s/network-policies/user-service.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: user-service
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: user-service
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: api-gateway
    ports:
    - protocol: TCP
      port: 3000
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
```

### Pod Security Policy

```yaml
# k8s/security/pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: restricted
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
  - ALL
  volumes:
  - 'configMap'
  - 'emptyDir'
  - 'projected'
  - 'secret'
  - 'downwardAPI'
  - 'persistentVolumeClaim'
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: true
```

---

## Rollback Procedures

### Kubernetes Rollback

```bash
# Rollback to previous version
kubectl rollout undo deployment/user-service -n production

# Rollback to specific revision
kubectl rollout undo deployment/user-service --to-revision=3 -n production

# Check rollout status
kubectl rollout status deployment/user-service -n production

# View rollout history
kubectl rollout history deployment/user-service -n production
```

### Database Rollback

```bash
# Flyway rollback (requires Flyway Teams)
flyway undo

# Manual rollback
psql -h $DB_HOST -U $DB_USER -d ecommerce < rollback-scripts/V2__undo.sql
```

---

## Cost Optimization

### Cluster Autoscaler

```yaml
# k8s/autoscaling/cluster-autoscaler.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-autoscaler
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cluster-autoscaler
  template:
    metadata:
      labels:
        app: cluster-autoscaler
    spec:
      serviceAccountName: cluster-autoscaler
      containers:
      - image: k8s.gcr.io/autoscaling/cluster-autoscaler:v1.27.0
        name: cluster-autoscaler
        command:
        - ./cluster-autoscaler
        - --v=4
        - --stderrthreshold=info
        - --cloud-provider=aws
        - --skip-nodes-with-local-storage=false
        - --expander=least-waste
        - --node-group-auto-discovery=asg:tag=k8s.io/cluster-autoscaler/enabled,k8s.io/cluster-autoscaler/ecommerce-production
        - --balance-similar-node-groups
        - --skip-nodes-with-system-pods=false
```

### Karpenter (Alternative to Cluster Autoscaler)

```yaml
# k8s/autoscaling/karpenter-provisioner.yaml
apiVersion: karpenter.sh/v1alpha5
kind: Provisioner
metadata:
  name: default
spec:
  requirements:
  - key: karpenter.sh/capacity-type
    operator: In
    values: ["spot", "on-demand"]
  - key: kubernetes.io/arch
    operator: In
    values: ["amd64"]
  - key: node.kubernetes.io/instance-type
    operator: In
    values: ["t3.medium", "t3.large", "t3.xlarge"]
  limits:
    resources:
      cpu: 1000
      memory: 1000Gi
  providerRef:
    name: default
  ttlSecondsAfterEmpty: 30
```

---

## Conclusion

This deployment guide provides a comprehensive foundation for deploying and managing the e-commerce platform. Regular updates and refinements ensure the deployment strategy evolves with the platform's needs.

Key takeaways:
- Infrastructure as Code for reproducibility
- Automated CI/CD for fast, reliable deployments
- Comprehensive monitoring and observability
- Strong security posture
- Disaster recovery preparedness
- Cost optimization strategies
