package procedures

// This is the main file to concatenate all queries that create a procedure,
// and export one string to be executed in db.go init function.

const CreateProcedures = createUserAndSession + createRepository + prepareFile + createFilePart
