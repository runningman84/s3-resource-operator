# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to the project maintainers. You can find the maintainer contact information in the repository settings.

Please include the following information:

- Type of vulnerability
- Full path of the source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

We will acknowledge your email within 48 hours and send a more detailed response within 7 days indicating the next steps.

## Security Best Practices

When deploying the S3 Resource Operator:

1. **Secrets Management**
   - Never commit credentials to version control
   - Use Kubernetes secrets or external secret managers (e.g., External Secrets Operator)
   - Rotate credentials regularly

2. **RBAC Configuration**
   - The operator requires cluster-wide secret read permissions
   - Review and restrict permissions according to your security policies
   - Use the provided ClusterRole and limit scope if needed

3. **Network Security**
   - Ensure S3 endpoints are accessed over secure connections (HTTPS)
   - Use network policies to restrict operator network access
   - Consider using private endpoints for S3 services

4. **Container Security**
   - The operator runs as a non-root user in the container
   - Regular security scans are performed using Trivy in CI/CD
   - Keep the operator updated to receive security patches

5. **Monitoring**
   - Monitor the operator's Prometheus metrics for unusual activity
   - Set up alerts for error rates and failed operations
   - Review logs regularly for suspicious patterns

## Security Updates

Security updates will be released as soon as possible after a vulnerability is confirmed. Updates will be:
- Released as patch versions
- Documented in the CHANGELOG
- Announced in GitHub releases

## Vulnerability Disclosure Timeline

1. Security issue reported
2. Acknowledgment within 48 hours
3. Investigation and fix development
4. Security advisory published
5. Patch released
6. Public disclosure (coordinated)
