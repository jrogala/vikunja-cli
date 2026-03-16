Feature: Project management
  As a user I can list projects via the CLI.

  Background:
    Given a running Vikunja instance
    And an authenticated user

  Scenario: List default projects
    When I list projects
    Then I should see "Inbox"

  Scenario: JSON output for projects
    When I list projects with JSON output
    Then the output should be valid JSON
