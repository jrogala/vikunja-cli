Feature: Setup configuration
  As a user I can configure the CLI to connect to my Vikunja instance.

  Background:
    Given a running Vikunja instance

  Scenario: Setup with valid credentials
    When I run setup with valid URL and token
    Then the config file should exist
    And the config file should contain the URL
    And the config file should contain the token

  Scenario: Setup with invalid URL
    When I run setup with invalid URL
    Then the output should contain "cannot reach Vikunja"

  Scenario: Setup with invalid token
    When I run setup with valid URL and invalid token
    Then the output should contain "token validation failed"

  Scenario: CLI works after setup without env vars
    When I run setup with valid URL and token
    And I list tasks without env vars
    Then the output should not contain "Error"
