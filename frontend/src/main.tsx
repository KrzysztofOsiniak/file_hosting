import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider, redirect } from "react-router-dom"
import { data } from "react-router"
import Error from './error.tsx'
import Login from './login.tsx'
import Signup from './signup.tsx'
import RootHeader from './rootheader.tsx'
import Home from './home.tsx'

const router = createBrowserRouter([
    {
        element: <RootHeader/>,
        errorElement: <Error/>,
        loader: rootHeaderLoader,
        children: [
            {
                path: "/",
                loader: function loader() {
                    return redirect('/home')
                },
            },
            {
                path: "/home",
                element: <Home/>,
                errorElement: <Error/>
            }
        ]
    },
    {
        path: "/login",
        element: <Login/>,
        errorElement: <Error/>,
        loader: userNotLoggedIn
    },
    {
        path: "/signup",
        element: <Signup/>,
        errorElement: <Error/>,
        loader: userNotLoggedIn
    }
])

async function userNotLoggedIn() {
    const res = await fetch("/api/user/account")
    if(res.status == 200) {
        return redirect("/home")
    }
}

async function rootHeaderLoader() {
    const res = await fetch("/api/user/account")
    if(res.status == 500) {
        throw data(res.status)
    }
    if(res.status != 200) {
        return {username: null, role: null, spaceTaken: null, space: null}
    }
    const body = await res.json()
    return {username: body.username, role: body.role, 
        freeSpace: body.space-body.spaceTaken, space: body.space}
}

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <RouterProvider router={router}/>
    </StrictMode>,
)
