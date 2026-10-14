# Project Rules

1. **No Comments**: Do not write comments in any source code files (except required declarative Swagger annotations on handlers/main).
2. **Package Restrictions**: Do not use any third-party packages except those explicitly specified in [plan.md](file:///home/arash/repo/ecommerce-server/plan.md).
3. **Architecture & Structure**: Adhere strictly to the project structure and architecture designated in [plan.md](file:///home/arash/repo/ecommerce-server/plan.md).
4. **Documentation & Verification**: Before writing any logic or code examples from packages, check their GitHub repository for the official and latest documentation first. Retrieve this documentation directly using `curl` from the local machine.
5. **Swagger Documentation for Handlers**: For every handler created, write its documented Swagger annotations. This includes:
   - Request DTOs (body) or path/query parameters based on the handler's requirements.
   - The success response DTO and status code (`@Success`). Do not add `@Failure` annotations.
   - Summary, tags, accepted content types, and route mapping (`@Router`).
do not test or build the app, ill do it my self