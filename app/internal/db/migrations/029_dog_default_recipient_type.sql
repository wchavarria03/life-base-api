-- Migration 029: a dog's usual/default recipient container type — needed
-- for the food<->recipients calculator to know what size portion each dog
-- normally gets, without requiring it to be picked every time.

alter table dogs
    add column default_recipient_type_id uuid references dog_recipient_types(id);
