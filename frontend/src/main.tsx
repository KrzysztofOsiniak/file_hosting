import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider, redirect } from "react-router-dom"
import Login, { userNotLoggedIn } from './login.tsx'
import Signup from './signup.tsx'
import RootHeader, { loader as rootHeaderLoader } from './rootheader.tsx'
import Home from './home.tsx'

const router = createBrowserRouter([
    {
        element: <RootHeader/>,
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
                element: <Home/>
            }
        ]
    },
    {
        path: "/login",
        element: <Login/>,
        loader: userNotLoggedIn
    },
    {
        path: "/signup",
        element: <Signup/>,
        loader: userNotLoggedIn
    }
])

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <RouterProvider router={router}/>
    </StrictMode>,
)
