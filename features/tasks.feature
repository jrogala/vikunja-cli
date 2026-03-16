Feature: Task management
  As a user I can manage tasks via the CLI.

  Background:
    Given a running Vikunja instance
    And an authenticated user

  Scenario: List tasks when empty
    When I list tasks
    Then I should see "No tasks found."

  Scenario: Create a task
    When I create a task "Buy groceries" in project 1
    Then the output should contain "Created task"
    And task "Buy groceries" should exist

  Scenario: Create a task with priority
    When I create a task "Fix critical bug" in project 1 with priority 4
    Then task "Fix critical bug" should have priority 4

  Scenario: Create a task with due date
    When I create a task "Submit report" in project 1 with due date "2026-12-31"
    Then task "Submit report" should have a due date

  Scenario: Complete a task
    Given a task "Deploy v2" exists in project 1
    When I complete the task "Deploy v2"
    Then task "Deploy v2" should be done

  Scenario: Edit task title
    Given a task "Old title" exists in project 1
    When I edit task "Old title" with --title "New title"
    Then task "New title" should exist
    And task "Old title" should not exist

  Scenario: Edit task priority
    Given a task "Prioritize me" exists in project 1
    When I edit task "Prioritize me" with --priority 4
    Then task "Prioritize me" should have priority 4

  Scenario: Edit task description
    Given a task "Describe me" exists in project 1
    When I edit task "Describe me" with --description "Added details"
    Then the output should contain "Updated task"

  Scenario: Mark task as undone
    Given a task "Reopen me" exists in project 1
    And task "Reopen me" is completed
    When I edit task "Reopen me" with --undone
    Then task "Reopen me" should not be done

  Scenario: Delete a task
    Given a task "Temp task" exists in project 1
    When I delete the task "Temp task"
    Then task "Temp task" should not exist

  Scenario: List tasks filters completed
    Given a task "Open task" exists in project 1
    And a task "Done task" exists in project 1
    And task "Done task" is completed
    When I list tasks
    Then I should see "Open task"
    And I should not see "Done task"

  Scenario: List all tasks includes completed
    Given a task "Active" exists in project 1
    And a task "Finished" exists in project 1
    And task "Finished" is completed
    When I list all tasks
    Then I should see "Active"
    And I should see "Finished"

  Scenario: List tasks by project
    Given a project "Work" exists
    And a task "Work item" exists in project "Work"
    When I list tasks in project "Work"
    Then I should see "Work item"

  Scenario: JSON output
    Given a task "JSON test" exists in project 1
    When I list tasks with JSON output
    Then the output should be valid JSON
    And the JSON should contain a task titled "JSON test"
