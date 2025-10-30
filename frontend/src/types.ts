export type RepositoryResponse = {
    name: string,
    members: Member[],
    files: File[],
    userPermission: "owner" | "none" |"full" | "read"
}
type Member = {
    id: number,
    username: string,
    permission: "" | "full" | "read"
}
type File = {
    id: number,
    ownerUsername: string,
    path: string,
    type: "file" | "folder",
    size: number,
    uploadDate: number
}
